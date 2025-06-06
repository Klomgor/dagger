package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/mod/semver"
	"golang.org/x/sync/errgroup"

	"github.com/BurntSushi/toml"
	"github.com/dagger/dagger/.dagger/internal/dagger"
)

const (
	rustGeneratedAPIPath  = "sdk/rust/crates/dagger-sdk/src/gen.rs"
	rustVersionFilePath   = "sdk/rust/crates/dagger-sdk/src/core/version.rs"
	rustCargoTomlFilePath = "sdk/rust/Cargo.toml"
	rustCargoLockFilePath = "sdk/rust/Cargo.lock"

	// https://hub.docker.com/_/rust
	rustDockerStable = "rust:1.77-bookworm"
	cargoChefVersion = "0.1.62"
	cargoEditVersion = "0.13.0"
)

type RustSDK struct {
	Dagger *DaggerDev // +private
}

// Lint the Rust SDK
func (r RustSDK) Lint(ctx context.Context) error {
	base := r.rustBase(rustDockerStable)

	eg := errgroup.Group{}
	eg.Go(func() error {
		_, err := base.
			WithExec([]string{"cargo", "check", "--all", "--release"}).
			Sync(ctx)
		return err
	})

	eg.Go(func() error {
		_, err := base.
			WithExec([]string{"cargo", "fmt", "--check"}).
			Sync(ctx)
		return err
	})

	eg.Go(func() error {
		before := r.Dagger.Source
		after, err := r.Generate(ctx)
		if err != nil {
			return err
		}
		return dag.Dirdiff().AssertEqual(ctx, before, after, []string{"sdk/rust"})
	})

	return eg.Wait()
}

// Test the Rust SDK
func (r RustSDK) Test(ctx context.Context) error {
	installer := r.Dagger.installer("sdk")
	_, err := r.rustBase(rustDockerStable).
		With(installer).
		WithExec([]string{"rustc", "--version"}).
		WithExec([]string{"cargo", "test", "--release", "--all"}).
		Sync(ctx)
	return err
}

// Regenerate the Rust SDK API
func (r RustSDK) Generate(ctx context.Context) (*dagger.Directory, error) {
	installer := r.Dagger.installer("sdk")
	introspection := r.Dagger.introspection(installer)
	generated := r.rustBase(rustDockerStable).
		WithMountedFile("/introspection.json", introspection).
		WithExec([]string{"cargo", "run", "-p", "dagger-bootstrap", "generate", "/introspection.json", "--output", fmt.Sprintf("/%s", rustGeneratedAPIPath)}).
		WithExec([]string{"cargo", "fix", "--all", "--allow-no-vcs"}).
		WithExec([]string{"cargo", "fmt"}).
		File(strings.TrimPrefix(rustGeneratedAPIPath, "sdk/rust/"))

	return dag.Directory().
		WithDirectory("sdk/rust", r.Dagger.Source.Directory("sdk/rust")).
		WithFile(rustGeneratedAPIPath, generated), nil
}

// Test the publishing process
func (r RustSDK) TestPublish(ctx context.Context, tag string) error {
	return r.Publish(ctx, tag, true, nil)
}

// Publish the Rust SDK
func (r RustSDK) Publish(
	ctx context.Context,
	tag string,

	// +optional
	dryRun bool,

	// +optional
	cargoRegistryToken *dagger.Secret,
) error {
	version := strings.TrimPrefix(tag, "sdk/rust/")

	versionFlag := strings.TrimPrefix(version, "v")
	if !semver.IsValid(version) {
		// just pick any version, it's a dry-run
		versionFlag = "--bump=rc"
	}

	crate := "dagger-sdk"
	base := r.
		rustBase(rustDockerStable).
		WithExec([]string{
			"cargo", "install", "cargo-edit@" + cargoEditVersion, "--locked",
		}).
		WithExec([]string{
			"cargo", "set-version", "-p", crate, versionFlag,
		})
	args := []string{
		"cargo", "publish", "-p", crate, "-v", "--all-features",
	}

	if dryRun {
		args = append(args, "--dry-run")
		base = base.WithExec(args)

		targetVersion := strings.TrimPrefix(version, "v")
		if !semver.IsValid(version) {
			cargoToml, err := base.File("Cargo.toml").Contents(ctx)
			if err != nil {
				return err
			}
			var config struct {
				Workspace struct {
					Package struct {
						Version string
					}
				}
			}
			_, err = toml.Decode(cargoToml, &config)
			if err != nil {
				return err
			}
			targetVersion = config.Workspace.Package.Version
		}

		// check we created the right files
		_, err := base.Directory(fmt.Sprintf("./target/package/dagger-sdk-%s", targetVersion)).Sync(ctx)
		if err != nil {
			return err
		}
		_, err = base.File(fmt.Sprintf("./target/package/dagger-sdk-%s.crate", targetVersion)).Sync(ctx)
		if err != nil {
			return err
		}

		// check that Cargo.toml got the version
		dt, err := base.File(fmt.Sprintf("./target/package/dagger-sdk-%s/Cargo.toml", targetVersion)).Contents(ctx)
		if err != nil {
			return err
		}
		if !strings.Contains(dt, "\nversion = \""+targetVersion+"\"\n") {
			//nolint:stylecheck
			return fmt.Errorf("Cargo.toml did not contain %q", targetVersion)
		}
	} else {
		base = base.
			WithSecretVariable("CARGO_REGISTRY_TOKEN", cargoRegistryToken).
			WithExec(args)
	}
	_, err := base.Sync(ctx)
	if err != nil {
		return err
	}

	return nil
}

// Bump the Rust SDK's Engine dependency
func (r RustSDK) Bump(ctx context.Context, version string) (*dagger.Directory, error) {
	versionStr := `pub const DAGGER_ENGINE_VERSION: &'static str = "([0-9\.-a-zA-Z]+)";`
	versionStrf := `pub const DAGGER_ENGINE_VERSION: &'static str = "%s";`
	version = strings.TrimPrefix(version, "v")

	versionContents, err := r.Dagger.Source.File(rustVersionFilePath).Contents(ctx)
	if err != nil {
		return nil, err
	}

	versionRe, err := regexp.Compile(versionStr)
	if err != nil {
		return nil, err
	}

	versionBumpedContents := versionRe.ReplaceAllString(
		versionContents,
		fmt.Sprintf(versionStrf, version),
	)

	crate := "dagger-sdk"
	base := r.
		rustBase(rustDockerStable).
		WithExec([]string{
			"cargo", "install", "cargo-edit@" + cargoEditVersion, "--locked",
		}).
		WithExec([]string{
			"cargo", "set-version", "-p", crate, version,
		})

	return dag.Directory().WithNewFile(rustVersionFilePath, versionBumpedContents).
		WithFile(rustCargoTomlFilePath, base.File("Cargo.toml")).
		WithFile(rustCargoLockFilePath, base.File("Cargo.lock")), nil
}

func (r RustSDK) rustBase(image string) *dagger.Container {
	const appDir = "sdk/rust"

	src := dag.Directory().WithDirectory("/", r.Dagger.Source.Directory(appDir))

	mountPath := fmt.Sprintf("/%s", appDir)

	base := dag.Container().
		From(image).
		WithDirectory(mountPath, src, dagger.ContainerWithDirectoryOpts{
			Include: []string{
				"**/Cargo.toml",
				"**/Cargo.lock",
				"**/main.rs",
				"**/lib.rs",
			},
		}).
		WithWorkdir(mountPath).
		WithEnvVariable("CARGO_HOME", "/root/.cargo").
		WithMountedCache("/root/.cargo", dag.CacheVolume("rust-cargo-"+image)).
		// combine into one layer so there's no assumptions on state of cache volume across steps
		WithExec([]string{"sh", "-c",
			strings.Join([]string{
				"rustup component add rustfmt",
				"cargo install --locked cargo-chef@" + cargoChefVersion,
				"cargo chef prepare --recipe-path /tmp/recipe.json",
				"cargo chef cook --release --workspace --recipe-path /tmp/recipe.json",
			}, "&& "),
		}).
		WithMountedDirectory(mountPath, src)

	return base
}

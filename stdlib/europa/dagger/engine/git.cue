package engine

// Push a directory to a git remote
#GitPush: {
	@dagger(notimplemented)
	$dagger: task: _name: "GitPush"

	input:  #FS
	remote: string
	ref:    string
}

// Pull a directory from a git remote
#GitPull: {
	$dagger: task: _name: "GitPull"
	remote: string
	ref:    string
	auth: {
		token?:  #Secret
		header?: #Secret
	}
	output: #FS
}

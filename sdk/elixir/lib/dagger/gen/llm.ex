# This file generated by `dagger_codegen`. Please DO NOT EDIT.
defmodule Dagger.LLM do
  @moduledoc """
  Dagger.LLM
  """

  use Dagger.Core.Base, kind: :object, name: "LLM"

  alias Dagger.Core.Client
  alias Dagger.Core.QueryBuilder, as: QB

  @derive Dagger.ID
  @derive Dagger.Sync
  defstruct [:query_builder, :client]

  @type t() :: %__MODULE__{}

  @doc """
  create a branch in the LLM's history
  """
  @spec attempt(t(), integer()) :: Dagger.LLM.t()
  def attempt(%__MODULE__{} = llm, number) do
    query_builder =
      llm.query_builder |> QB.select("attempt") |> QB.put_arg("number", number)

    %Dagger.LLM{
      query_builder: query_builder,
      client: llm.client
    }
  end

  @doc """
  returns the type of the current state
  """
  @spec bind_result(t(), String.t()) :: Dagger.Binding.t() | nil
  def bind_result(%__MODULE__{} = llm, name) do
    query_builder =
      llm.query_builder |> QB.select("bindResult") |> QB.put_arg("name", name)

    %Dagger.Binding{
      query_builder: query_builder,
      client: llm.client
    }
  end

  @doc """
  return the LLM's current environment
  """
  @spec env(t()) :: Dagger.Env.t()
  def env(%__MODULE__{} = llm) do
    query_builder =
      llm.query_builder |> QB.select("env")

    %Dagger.Env{
      query_builder: query_builder,
      client: llm.client
    }
  end

  @doc """
  return the llm message history
  """
  @spec history(t()) :: {:ok, [String.t()]} | {:error, term()}
  def history(%__MODULE__{} = llm) do
    query_builder =
      llm.query_builder |> QB.select("history")

    Client.execute(llm.client, query_builder)
  end

  @doc """
  return the raw llm message history as json
  """
  @spec history_json(t()) :: {:ok, Dagger.JSON.t()} | {:error, term()}
  def history_json(%__MODULE__{} = llm) do
    query_builder =
      llm.query_builder |> QB.select("historyJSON")

    Client.execute(llm.client, query_builder)
  end

  @doc """
  A unique identifier for this LLM.
  """
  @spec id(t()) :: {:ok, Dagger.LLMID.t()} | {:error, term()}
  def id(%__MODULE__{} = llm) do
    query_builder =
      llm.query_builder |> QB.select("id")

    Client.execute(llm.client, query_builder)
  end

  @doc """
  return the last llm reply from the history
  """
  @spec last_reply(t()) :: {:ok, String.t()} | {:error, term()}
  def last_reply(%__MODULE__{} = llm) do
    query_builder =
      llm.query_builder |> QB.select("lastReply")

    Client.execute(llm.client, query_builder)
  end

  @doc """
  synchronize LLM state
  """
  @spec loop(t()) :: Dagger.LLM.t()
  def loop(%__MODULE__{} = llm) do
    query_builder =
      llm.query_builder |> QB.select("loop")

    %Dagger.LLM{
      query_builder: query_builder,
      client: llm.client
    }
  end

  @doc """
  return the model used by the llm
  """
  @spec model(t()) :: {:ok, String.t()} | {:error, term()}
  def model(%__MODULE__{} = llm) do
    query_builder =
      llm.query_builder |> QB.select("model")

    Client.execute(llm.client, query_builder)
  end

  @doc """
  return the provider used by the llm
  """
  @spec provider(t()) :: {:ok, String.t()} | {:error, term()}
  def provider(%__MODULE__{} = llm) do
    query_builder =
      llm.query_builder |> QB.select("provider")

    Client.execute(llm.client, query_builder)
  end

  @doc """
  synchronize LLM state
  """
  @spec sync(t()) :: {:ok, Dagger.LLM.t()} | {:error, term()}
  def sync(%__MODULE__{} = llm) do
    query_builder =
      llm.query_builder |> QB.select("sync")

    with {:ok, id} <- Client.execute(llm.client, query_builder) do
      {:ok,
       %Dagger.LLM{
         query_builder:
           QB.query()
           |> QB.select("loadLLMFromID")
           |> QB.put_arg("id", id),
         client: llm.client
       }}
    end
  end

  @doc """
  returns the token usage of the current state
  """
  @spec token_usage(t()) :: Dagger.LLMTokenUsage.t()
  def token_usage(%__MODULE__{} = llm) do
    query_builder =
      llm.query_builder |> QB.select("tokenUsage")

    %Dagger.LLMTokenUsage{
      query_builder: query_builder,
      client: llm.client
    }
  end

  @doc """
  print documentation for available tools
  """
  @spec tools(t()) :: {:ok, String.t()} | {:error, term()}
  def tools(%__MODULE__{} = llm) do
    query_builder =
      llm.query_builder |> QB.select("tools")

    Client.execute(llm.client, query_builder)
  end

  @doc """
  allow the LLM to interact with an environment via MCP
  """
  @spec with_env(t(), Dagger.Env.t()) :: Dagger.LLM.t()
  def with_env(%__MODULE__{} = llm, env) do
    query_builder =
      llm.query_builder |> QB.select("withEnv") |> QB.put_arg("env", Dagger.ID.id!(env))

    %Dagger.LLM{
      query_builder: query_builder,
      client: llm.client
    }
  end

  @doc """
  swap out the llm model
  """
  @spec with_model(t(), String.t()) :: Dagger.LLM.t()
  def with_model(%__MODULE__{} = llm, model) do
    query_builder =
      llm.query_builder |> QB.select("withModel") |> QB.put_arg("model", model)

    %Dagger.LLM{
      query_builder: query_builder,
      client: llm.client
    }
  end

  @doc """
  append a prompt to the llm context
  """
  @spec with_prompt(t(), String.t()) :: Dagger.LLM.t()
  def with_prompt(%__MODULE__{} = llm, prompt) do
    query_builder =
      llm.query_builder |> QB.select("withPrompt") |> QB.put_arg("prompt", prompt)

    %Dagger.LLM{
      query_builder: query_builder,
      client: llm.client
    }
  end

  @doc """
  append the contents of a file to the llm context
  """
  @spec with_prompt_file(t(), Dagger.File.t()) :: Dagger.LLM.t()
  def with_prompt_file(%__MODULE__{} = llm, file) do
    query_builder =
      llm.query_builder |> QB.select("withPromptFile") |> QB.put_arg("file", Dagger.ID.id!(file))

    %Dagger.LLM{
      query_builder: query_builder,
      client: llm.client
    }
  end

  @doc """
  Add a system prompt to the LLM's environment
  """
  @spec with_system_prompt(t(), String.t()) :: Dagger.LLM.t()
  def with_system_prompt(%__MODULE__{} = llm, prompt) do
    query_builder =
      llm.query_builder |> QB.select("withSystemPrompt") |> QB.put_arg("prompt", prompt)

    %Dagger.LLM{
      query_builder: query_builder,
      client: llm.client
    }
  end

  @doc """
  Disable the default system prompt
  """
  @spec without_default_system_prompt(t()) :: Dagger.LLM.t()
  def without_default_system_prompt(%__MODULE__{} = llm) do
    query_builder =
      llm.query_builder |> QB.select("withoutDefaultSystemPrompt")

    %Dagger.LLM{
      query_builder: query_builder,
      client: llm.client
    }
  end
end

defimpl Jason.Encoder, for: Dagger.LLM do
  def encode(llm, opts) do
    {:ok, id} = Dagger.LLM.id(llm)
    Jason.Encode.string(id, opts)
  end
end

defimpl Nestru.Decoder, for: Dagger.LLM do
  def decode_fields_hint(_struct, _context, id) do
    {:ok, Dagger.Client.load_llm_from_id(Dagger.Global.dag(), id)}
  end
end

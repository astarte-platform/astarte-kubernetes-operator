defmodule Doc.MixProject do
  use Mix.Project

  @source_ref "release-23.5"
  @source_version String.replace_prefix(@source_ref, "release-", "")
                  |> String.replace("master", "snapshot")

  def project do
    [
      app: :doc,
      version: "23.5.0",
      elixir: "~> 1.14",
      start_permanent: Mix.env() == :prod,
      deps: deps(),
      name: "Astarte Operator",
      homepage_url:
        "https://docs.astarte-platform.org/astarte-kubernetes-operator/#{@source_version}/",
      source_url: "https://github.com/astarte-platform/astarte-kubernetes-operator",
      docs: docs()
    ]
  end

  defp deps do
    [{:ex_doc, "~> 0.29", only: :dev}]
  end

  # Add here additional documentation files
  defp docs do
    [
      main: "001-intro_administrator",
      logo: "images/mascot.png",
      javascript_config_path: "../common_vars.js",
      extras: Path.wildcard("pages/*/*.{cheatmd,md}"),
      assets: "images/",
      api_reference: false,
      source_ref: "#{@source_ref}/docs/documentation",
      groups_for_extras: [
        "Administrator Guide": ~r"/administrator/",
        "Upgrade Guide": ~r"/upgrade/",
        "CRDs Reference": ~r"/crds/"
      ]
    ]
  end
end

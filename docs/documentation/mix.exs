defmodule Doc.MixProject do
  use Mix.Project

  def project do
    [
      app: :doc,
      version: "23.05.0-dev",
      elixir: "~> 1.4",
      start_permanent: Mix.env() == :prod,
      deps: deps(),
      name: "Astarte Operator",
      homepage_url: "http://astarte-platform.org",
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
      source_url: "https://git.ispirata.com/Astarte-NG/%{path}#L%{line}",
      # It's in the docs repo root
      # TODO define the file in docs repo
      # javascript_config_path: "../astarte_operator_common_vars.js",
      extras: Path.wildcard("pages/*/*.{cheatmd,md}"),
      assets: "images/",
      api_reference: false,
      groups_for_extras: [
        "Administrator Guide": ~r"/administrator/",
        "Upgrade Guide": ~r"/upgrade/",
        "CRDs Reference": ~r"/crds/"
      ]
    ]
  end
end

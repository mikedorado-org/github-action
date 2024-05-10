> **Warning**
> ### This example repo was decommissioned and archived due to a change in [Actions Platform Strategy](https://github.com/github/compute-products/discussions/389).
> ### Please see:
> * Other Actions services as an example, such as https://github.com/github/actions-run-service and https://github.com/github/actions-broker
> * https://github.com/github/go-sample-service for the GitHub-wide sample Go service
> * https://github.com/github/go for GitHub-supported and recommended libraries
> * https://thehub.github.com/epd/engineering/dev-practicals/service-lifecycle for guidance on new service development and new service checklists

---

# actions-example-go

A repo that exemplifies the guidelines and best practices for Actions four nines services.


## 🚀 Developing

[![Build on Codespaces](https://github.com/codespaces/badge.svg)](https://github.com/codespaces/new?hide_repo_select=true&ref=main&repo=482604714)

Some common [devloops](https://github.com/github/actions-kupl-devex/blob/main/docs/devex_tools.md)

1. Build and deploy
```
./kupl server
```
2. Deploy on local changes
```
./kupl dev
```
3. Debug (using the `launch.json` file)
```
./kupl debug
```

## 🚀 Deploying...

`actions-example-go` deploys [to Moda](https://moda.githubapp.com/apps/actions-example-go)

1. To deploy to Moda from a PR, use `.deploy <PR>`. Or `.deploy <PR> to lab` to deploy the lab environment.
1. [auto-deploys](https://github.com/github/heaven/blob/master/docs/auto_deploys.md) is also enabled for the production environment. Once a commit is merged into main, the CI workflow will kick off and deploy.
1. To deploy the current branch to an environment, use `.deploy actions-example-go/main to <environment>`

## 📡 Telemetry

## 🪵 Logs

Logs are [in Kusto](https://kusto.azure.com/clusters/octokus2.eastus2/databases/Actions?query=H4sIAAAAAAAAA0tMLsnMzyvmqlEoz0gtSlUoycxNLS5JzC1QsFNITM/XMMwo0oRLZlsU+yUC5QsSk1NBDAVbWwWlRIgJuqkVQF05qbrp+boFRfkppWBhJS4FEMgvIlpvTmKSEgC5cG3vlQAAAA==) and Splunk

## 📏 Metrics

[DataDog metrics](https://app.datadoghq.com/metric/summary?tags=kube_service%3Aactions-example-go)

[Moda k8s dashboard](https://app.datadoghq.com/screen/integration/kubernetes?tpl_var_namespace%5B0%5D=actions-example-go-production)

## Contributing

For info on how to contribute to this repo please check [contributing](CONTRIBUTING.md).

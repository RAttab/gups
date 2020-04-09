# `gups` Helm Chart

A Helm chart for `gups`.

## Usage

This chart is meant to be used in a wrapper chart.

## Testing

To test your CronJob, use the following:

```shell
K8S_JOBNAME="gups-test-${USER}-$(date '+%Y%m%dT%H%M%S')"

kubectl create job --from=cronjob/{{ gups.fullname }} "${K8S_JOBNAME}"

kubectl get jobs --selector=job-name="${K8S_JOBNAME}"
kubectl get pods --selector=job-name="${K8S_JOBNAME}"
```

## Contributing

1.  Create your feature branch: `git checkout -b feature/my-new-feature`
2.  Commit your changes: `git commit -am 'Add some feature'`
3.  Push to the branch: `git push origin my-new-feature`
4.  Submit a pull request :D

## License

> Copyright AdGear | Samsung Ads

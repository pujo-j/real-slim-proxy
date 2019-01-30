# real-slim-proxy
SideCar dependency caching proxy with object-store backup for container based CI/CD

## Usage

It can either be used as daemon collocated with the build, or *better* as a sidecar proxy to a container.

The command line is as simple as can be, without arguments, it will search for a config file (slim.yaml) in the current working directory, if that's not to your liking use the --config flag to point to your config.

## Examples

### Java build in Google Container Engine

This is the original use case, building JVM applications in ephemeral containers, while it is always possible to deploy a nexus or an artifactory in a deploymentset, configuration, security, storage customisation is expensive, and both products are ... well, not lightweight.

#### Configure your environment

1. Create a bucket, and the required service account to access it

2. Inject the service account into a kubernetes secret

3. Inject the config into a kubernetes configmap

#### Do the actual building

1. Prepare your pod specification

2. Run once and appreciates that it works

3. Run twice and love the speed !


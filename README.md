# NGINX Server Cloud Native Buildpack

The NGINX CNB provides the [NGINX](https://www.nginx.com/) binary distribution. The buildpack installs
the NGINX binary distribution onto the `$PATH` which makes it available for
subsequent buildpacks.

## Integration

The NGINX CNB provides nginx as a dependency. Downstream buildpacks, like
[PHP Web CNB](https://github.com/paketo-buildpacks/php-web) can require the nginx
dependency by generating a [Build Plan
TOML](https://github.com/buildpacks/spec/blob/master/buildpack.md#build-plan-toml)
file that looks like the following:

```toml
[[requires]]

  # The name of the NGINX dependency is "nginx". This value is considered
  # part of the public API for the buildpack and will not change without a plan
  # for deprecation.
  name = "nginx"

  # The version of the NGINX dependency is not required. In the case it
  # is not specified, the buildpack will provide the default version, which can
  # be seen in the buildpack.toml file.
  # If you wish to request a specific version, the buildpack supports
  # specifying a semver constraint in the form of "1.*", "1.17.*", or even
  # "1.17.9".
  version = "1.17.9"

  # The NGINX buildpack supports some non-required metadata options.
  [requires.metadata]

    # Setting the launch flag to true will ensure that the NGINX
    # dependency is available on the $PATH for the running application. If you are
    # writing an application that needs to run NGINX at runtime, this flag should
    # be set to true.
    launch = true
```

## Usage

To package this buildpack for consumption:

```
$ ./scripts/package.sh
```

This builds the buildpack's Go source using `GOOS=linux` by default. You can supply another value as the first argument to `package.sh`.


## `buildpack.yml` Configurations

```yaml
nginx:
  # this allows you to specify a version constaint for the `NGINX` dependency
  # any valid semver constaints (e.g. 1.* and 1.17.*) are also acceptable
  #
  # you can also specify "mainline" or "stable"
  version: 1.17.9
```

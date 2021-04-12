# NGINX Server Cloud Native Buildpack

The NGINX buildpack provides the [NGINX](https://www.nginx.com/) binary distribution.
The buildpack installs the NGINX binary distribution onto the `$PATH` which
makes it available for subsequent buildpacks and/or the application image.

#### The NGINX buildpack is compatible with the following builder(s):
- [Paketo Full Builder](https://github.com/paketo-buildpacks/full-builder)
- [Paketo Base Builder](https://github.com/paketo-buildpacks/base-builder)

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

## Data driven templates

The NGINX buildpack supports data driven templates for nginx config. You can
use templated variables like `{{port}}`, `{{env "FOO"}}` and `{{module
"ngx_stream_module"}}` in your `nginx.conf` to use values known at launch time.

A usage example can be found in the [`samples` repository under the `nginx`
directory](https://github.com/paketo-buildpacks/samples/tree/main/nginx).

#### PORT

Use `{{port}}` to dynamically set the port at which the server will accepts requests. At launch time, the buildpack will read the value of `$PORT` to set the value of `{{port}}`.

For example, to set an NGINX server to listen on `$PORT`, use the following in your `nginx.conf` file:

```
server {
  listen {{port}};
}
```

Then run the built image using the `PORT` variable set as follows:

```
docker run --tty --env PORT=8080 --publish 8080:8080 my-nginx-image
```

#### Environment Variables

This is a generic case of the `{{port}}` directive described ealier. To use the
value of any environment variable `$FOOVAR` available at launch time, use the
directive `{{env "FOOVAR"}}` in your `nginx.conf`.

For example, include the following in your `nginx.conf` file to enable or
disable gzipping of responses based on the value of `GZIP_DOWNLOADS`:

```
gzip {{env "GZIP_DOWNLOADS"}};
```

Then run the built image using the `GZIP_DOWNLOADS` variable set as follows:

```
docker run --tty --env PORT=8080 --env GZIP_DOWNLOADS=off --publish 8080:8080 my-nginx-image
```

#### Loading dynamic modules

You can use templates to set the path to a dynamic module using the
`load_module` directive.

* To load a user-provided module named `ngx_foo_module`, provide a
  `modules/ngx_foo_module.so` file in your app directory and add the following
  to the top of your `nginx.conf` file:

```
{{module "ngx_foo_module"}}
```

* To load a buildpack-provided module like `ngx_stream_module`, add the
  following to the top of your `nginx.conf` file. You do not need to provide an
  `ngx_stream_module.so` file:

```
{{module "ngx_stream_module"}}
```

## Configurations

Specifying the NGINX Server version through `buildpack.yml` configuration
is deprecated and will not be supported in Apache HTTP Server Buildpack v1.0.0.

To migrate from using `buildpack.yml` please set the following environment
variables at build time either directly (ex. `pack build my-app --env
BP_ENVIRONMENT_VARIABLE=some-value`) or through a [`project.toml`
file](https://github.com/buildpacks/spec/blob/main/extensions/project-descriptor.md)

### `BP_NGINX_VERSION`
The `BP_NGINX_VERSION` variable allows you to specify the version of NGINX Server that is installed.

```shell
BP_NGINX_VERSION=1.19.8
```

This will replace the following structure in `buildpack.yml`:
```yaml
nginx:
  # this allows you to specify a version constraint for the nginx dependency
  # any valid semver constraints (e.g. 1.* and 1.19.*) are also acceptable
  version: "1.19.8"
```

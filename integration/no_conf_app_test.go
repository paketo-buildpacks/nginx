package integration_test

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/occam"

	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testNoConfApp(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker

		name            string
		source          string
		image           occam.Image
		configContainer occam.Container
		container       occam.Container
	)

	it.Before(func() {
		pack = occam.NewPack().WithNoColor().WithVerbose()

		docker = occam.NewDocker()

		var err error
		name, err = occam.RandomName()
		Expect(err).NotTo(HaveOccurred())

		source, err = occam.Source(filepath.Join("testdata", "no_conf_app"))
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(docker.Container.Remove.Execute(configContainer.ID)).To(Succeed())
		Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
		Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
		Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
		Expect(os.RemoveAll(source)).To(Succeed())
	})

	context("when pushing app with no conf and $BP_WEB_SERVER=nginx", func() {
		it("generates the default nginx.conf", func() {
			var err error
			image, _, err = pack.Build.
				WithBuildpacks(nginxBuildpack).
				WithEnv(map[string]string{
					"BP_WEB_SERVER": "nginx",
				}).
				WithPullPolicy("never").
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			configContainer, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				WithEntrypoint("launcher").
				WithCommand(`bash -c "cat /workspace/nginx.conf"`).
				Execute(image.ID)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() (string, error) {
				cLogs, err := docker.Container.Logs.Execute(configContainer.ID)
				if err != nil {
					return "", err
				}
				return cLogs.String(), nil
			}).Should(Equal(expectedDefaultConfig))

			container, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				Execute(image.ID)
			Expect(err).ToNot(HaveOccurred())

			Eventually(container).Should(Serve(ContainSubstring("<p>Hello World!</p>")).OnPort(8080))
		})
	})

	context("when using env var configuration options", func() {
		it("generates an nginx.conf with the configuration", func() {
			var err error
			image, _, err = pack.Build.
				WithBuildpacks(nginxBuildpack).
				WithEnv(map[string]string{
					"BP_WEB_SERVER":                   "nginx",
					"BP_WEB_SERVER_ROOT":              "custom_root",
					"BP_WEB_SERVER_ENABLE_PUSH_STATE": "true",
				}).
				WithPullPolicy("never").
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			configContainer, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				WithEntrypoint("launcher").
				WithCommand(`bash -c "cat /workspace/nginx.conf"`).
				Execute(image.ID)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() (string, error) {
				cLogs, err := docker.Container.Logs.Execute(configContainer.ID)
				if err != nil {
					return "", err
				}
				return cLogs.String(), nil
			}).Should(ContainSubstring(expectedPushStateConfig))

			Eventually(func() (string, error) {
				cLogs, err := docker.Container.Logs.Execute(configContainer.ID)
				if err != nil {
					return "", err
				}
				return cLogs.String(), nil
			}).Should(ContainSubstring(expectedCustomRootConfig))

			container, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				Execute(image.ID)
			Expect(err).ToNot(HaveOccurred())

			Eventually(container).Should(Serve(ContainSubstring("<p>Hello World!</p>")).OnPort(8080))
			Eventually(container).Should(Serve(ContainSubstring("<p>Hello World!</p>")).OnPort(8080).WithEndpoint("/test"))
		})
	})

	context("building with no config and forcing HTTPS connections", func() {
		it("generates an nginx.conf with the required redirect logic", func() {
			var err error
			image, _, err = pack.Build.
				WithBuildpacks(nginxBuildpack).
				WithEnv(map[string]string{
					"BP_WEB_SERVER":             "nginx",
					"BP_WEB_SERVER_ROOT":        "custom_root",
					"BP_WEB_SERVER_FORCE_HTTPS": "true",
				}).
				WithPullPolicy("never").
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			configContainer, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				WithEntrypoint("launcher").
				WithCommand(`bash -c "cat /workspace/nginx.conf"`).
				Execute(image.ID)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() (string, error) {
				cLogs, err := docker.Container.Logs.Execute(configContainer.ID)
				if err != nil {
					return "", err
				}
				return cLogs.String(), nil
			}).Should(ContainSubstring(expectedHTTPSConfig))

			container, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				Execute(image.ID)
			Expect(err).ToNot(HaveOccurred())

			client := &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}

			response, err := client.Get(fmt.Sprintf("http://localhost:%s", container.HostPort("8080")))
			Expect(err).NotTo(HaveOccurred())
			defer response.Body.Close()

			Expect(response.StatusCode).To(Equal(http.StatusMovedPermanently))

			_, err = http.Get(fmt.Sprintf("http://localhost:%s", container.HostPort("8080")))
			// Assert that the server attempts to hit HTTPS URL instead of HTTP
			Expect(err).To(MatchError(`Get "https://localhost/": dial tcp [::1]:443: connect: connection refused`))
		})
	})
}

var expectedCustomRootConfig string = `  server {
    listen 8080 default_server;
    server_name _;

    # Directory where static files are located
    root /workspace/custom_root;

`
var expectedHTTPSConfig string = `  server {
    listen 8080 default_server;
    server_name _;

    # Directory where static files are located
    root /workspace/custom_root;

    # If HTTP request is made, redirect to HTTPS requests
    set $updated_host $host;
    if ($http_x_forwarded_host != "") {
      set $updated_host $http_x_forwarded_host;
    }

    if ($http_x_forwarded_proto != "https") {
      return 301 https://$updated_host$request_uri;
    }
`

var expectedPushStateConfig string = `    location / {
      # Send the content at / in response to *any* requested endpoint
      if (!-e $request_filename) {
        rewrite ^(.*)$ / break;
      }
`

var expectedDefaultConfig string = `# Number of worker processes running in container
worker_processes 1;

# Run NGINX in foreground (necessary for containerized NGINX)
daemon off;

# Set the location of the server's PID file
pid /tmp/nginx.pid;

# Set the location of the server's error log
error_log stderr;

events {
  # Set number of simultaneous connections each worker process can serve
  worker_connections 1024;
}

http {
  client_body_temp_path /tmp/client_body_temp;
  proxy_temp_path /tmp/proxy_temp;
  fastcgi_temp_path /tmp/fastcgi_temp;

  charset utf-8;

  # Map media types to file extensions
  types {
    text/html html htm shtml;
    text/css css;
    text/xml xml;
    image/gif gif;
    image/jpeg jpeg jpg;
    application/javascript js;
    application/atom+xml atom;
    application/rss+xml rss;
    font/ttf ttf;
    font/woff woff;
    font/woff2 woff2;
    text/mathml mml;
    text/plain txt;
    text/vnd.sun.j2me.app-descriptor jad;
    text/vnd.wap.wml wml;
    text/x-component htc;
    text/cache-manifest manifest;
    image/png png;
    image/tiff tif tiff;
    image/vnd.wap.wbmp wbmp;
    image/x-icon ico;
    image/x-jng jng;
    image/x-ms-bmp bmp;
    image/svg+xml svg svgz;
    image/webp webp;
    application/java-archive jar war ear;
    application/mac-binhex40 hqx;
    application/msword doc;
    application/pdf pdf;
    application/postscript ps eps ai;
    application/rtf rtf;
    application/vnd.ms-excel xls;
    application/vnd.ms-powerpoint ppt;
    application/vnd.wap.wmlc wmlc;
    application/vnd.google-earth.kml+xml  kml;
    application/vnd.google-earth.kmz kmz;
    application/x-7z-compressed 7z;
    application/x-cocoa cco;
    application/x-java-archive-diff jardiff;
    application/x-java-jnlp-file jnlp;
    application/x-makeself run;
    application/x-perl pl pm;
    application/x-pilot prc pdb;
    application/x-rar-compressed rar;
    application/x-redhat-package-manager  rpm;
    application/x-sea sea;
    application/x-shockwave-flash swf;
    application/x-stuffit sit;
    application/x-tcl tcl tk;
    application/x-x509-ca-cert der pem crt;
    application/x-xpinstall xpi;
    application/xhtml+xml xhtml;
    application/zip zip;
    application/octet-stream bin exe dll;
    application/octet-stream deb;
    application/octet-stream dmg;
    application/octet-stream eot;
    application/octet-stream iso img;
    application/octet-stream msi msp msm;
    application/json json;
    audio/midi mid midi kar;
    audio/mpeg mp3;
    audio/ogg ogg;
    audio/x-m4a m4a;
    audio/x-realaudio ra;
    video/3gpp 3gpp 3gp;
    video/mp4 mp4;
    video/mpeg mpeg mpg;
    video/quicktime mov;
    video/webm webm;
    video/x-flv flv;
    video/x-m4v m4v;
    video/x-mng mng;
    video/x-ms-asf asx asf;
    video/x-ms-wmv wmv;
    video/x-msvideo avi;
  }

  access_log /dev/stdout;

  # Set the default MIME type of responses; 'application/octet-stream'
  # represents an arbitrary byte stream
  default_type application/octet-stream;

  # (Performance) When sending files, skip copying into buffer before sending.
  sendfile on;
  # (Only active with sendfile on) wait for packets to reach max size before
  # sending.
  tcp_nopush on;

  # (Performance) Enable compressing responses
  gzip on;
  # For all clients
  gzip_static always;
  # Including responses to proxied requests
  gzip_proxied any;
  # For responses above a certain length
  gzip_min_length 1100;
  # That are one of the following MIME types
  gzip_types text/plain text/css text/js text/xml text/javascript application/javascript application/x-javascript application/json application/xml application/xml+rss;
  # Compress responses to a medium degree
  gzip_comp_level 6;
  # Using 16 buffers of 8k bytes each
  gzip_buffers 16 8k;

  # Add "Vary: Accept-Encoding” response header to compressed responses
  gzip_vary on;

  # Decompress responses if client doesn't support compressed
  gunzip on;

  # Don't compress responses if client is Internet Explorer 6
  gzip_disable "msie6";

  # Set a timeout during which a keep-alive client connection will stay open on
  # the server side
  # TODO: Consider downgrading to 15 seconds
  keepalive_timeout 30;

  # Ensure that redirects don't include the internal container PORT - <%=
  # ENV["PORT"] %>
  port_in_redirect off;

  # (Security) Disable emitting nginx version on error pages and in the
  # “Server” response header field
  server_tokens off;

  server {
    listen 8080 default_server;
    server_name _;

    # Directory where static files are located
    root /workspace/public;

    location / {
      # Specify files sent to client if specific file not requested (e.g.
      # GET www.example.com/). NGINX sends first existing file in the list.
      index index.html index.htm Default.htm;
    }

    # (Security) Don't serve dotfiles, except .well-known/, which is needed by
    # LetsEncrypt
    location ~ /\.(?!well-known) {
      deny all;
      return 404;
    }
  }
}
`

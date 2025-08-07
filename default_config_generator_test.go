package nginx_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/nginx"
	"github.com/paketo-buildpacks/occam/matchers"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/sclevine/spec"
)

func testDefaultConfigGenerator(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string
		buffer     *bytes.Buffer
		logs       scribe.Emitter
		generator  nginx.DefaultConfigGenerator
	)

	it.Before(func() {
		workingDir = t.TempDir()

		buffer = bytes.NewBuffer(nil)
		logs = scribe.NewEmitter(buffer)

		generator = nginx.NewDefaultConfigGenerator(logs)
	})

	context("Generate", func() {
		it("writes a default nginx.conf to the working directory", func() {
			err := generator.Generate(nginx.Configuration{
				NGINXConfLocation: filepath.Join(workingDir, "nginx.conf"),
				WebServerRoot:     "./public",
			})
			Expect(err).NotTo(HaveOccurred())

			topLevelContext := ContainSubstring(`# Number of worker processes running in container
worker_processes 1;

# Run NGINX in foreground (necessary for containerized NGINX)
daemon off;

# Set the location of the server's error log
error_log stderr;

`)

			eventsContext := ContainSubstring(`events {
  # Set number of simultaneous connections each worker process can serve
  worker_connections 1024;
}

`)

			tempFileLocations := ContainSubstring(`
  client_body_temp_path {{ tempDir }}/client_body_temp;
  proxy_temp_path {{ tempDir }}/proxy_temp;
  fastcgi_temp_path {{ tempDir }}/fastcgi_temp;

`)

			mediaTypeToFileExtensionMapping := ContainSubstring(`  types {
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

`)

			logToStdOut := ContainSubstring(` access_log /dev/stdout;`)

			defaultResponseMimeType := ContainSubstring(`  default_type application/octet-stream;`)

			performanceEnhancementsForPageLoadSpeed := ContainSubstring(`  # (Performance) When sending files, skip copying into buffer before sending.
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
  gzip_types
    text/plain
    text/css
    text/js
    text/xml
    text/javascript
    application/javascript
    application/x-javascript
    application/json
    application/xml
    application/xml+rss
    font/eot
    font/otf
    font/ttf
    image/svg+xml;
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
`)

			connectionTimeout := ContainSubstring(`  keepalive_timeout 30;`)

			excludeContainerPortInRedirects := ContainSubstring(`  port_in_redirect off;`)

			excludeNginxServerInfoInResponses := ContainSubstring(`  server_tokens off;`)

			defaultServerContext := ContainSubstring(`  server {
    listen {{port}} default_server;
    server_name _;

    # Directory where static files are located
    root {{ env "APP_ROOT" }}/public;

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
`)

			Expect(filepath.Join(workingDir, "nginx.conf")).To(matchers.BeAFileMatching(And(
				topLevelContext,
				eventsContext,
				tempFileLocations,
				mediaTypeToFileExtensionMapping,
				logToStdOut,
				defaultResponseMimeType,
				performanceEnhancementsForPageLoadSpeed,
				connectionTimeout,
				excludeContainerPortInRedirects,
				excludeNginxServerInfoInResponses,
				defaultServerContext,
			)))
		})

		it("writes a nginx.conf with specified relative root directory", func() {
			err := generator.Generate(nginx.Configuration{
				NGINXConfLocation: filepath.Join(workingDir, "nginx.conf"),
				WebServerRoot:     "custom",
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(filepath.Join(workingDir, "nginx.conf")).
				To(matchers.BeAFileMatching(ContainSubstring(`root {{ env "APP_ROOT" }}/custom;`)))
		})

		it("writes a nginx.conf with specified absolute path to root directory", func() {
			err := generator.Generate(nginx.Configuration{
				NGINXConfLocation: filepath.Join(workingDir, "nginx.conf"),
				WebServerRoot:     "/some/absolute/path",
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(filepath.Join(workingDir, "nginx.conf")).
				To(matchers.BeAFileMatching(ContainSubstring(`root /some/absolute/path;`)))
		})

		it("writes a nginx.conf with specified location path", func() {
			err := generator.Generate(nginx.Configuration{
				NGINXConfLocation:     filepath.Join(workingDir, "nginx.conf"),
				WebServerLocationPath: "/path",
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(filepath.Join(workingDir, "nginx.conf")).
				To(matchers.BeAFileMatching(ContainSubstring(`location /path {`)))
		})

		it("writes a nginx.conf including custom file", func() {
			err := generator.Generate(nginx.Configuration{
				NGINXConfLocation:        filepath.Join(workingDir, "nginx.conf"),
				WebServerLocationPath:    "/path",
				WebServerIncludeFilePath: "./custom-include.conf",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(filepath.Join(workingDir, "nginx.conf")).
				To(matchers.BeAFileMatching(ContainSubstring(`include ./custom-include.conf;`)))
		})

		it("writes an nginx.conf that conditionally includes the PushState content", func() {
			err := generator.Generate(nginx.Configuration{
				NGINXConfLocation:        filepath.Join(workingDir, "nginx.conf"),
				WebServerEnablePushState: true,
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(filepath.Join(workingDir, "nginx.conf")).
				To(matchers.BeAFileMatching(ContainSubstring(`    location / {
      # Send the content at / in response to *any* requested endpoint
      if (!-e $request_filename) {
        rewrite ^(.*)$ / break;
      }
`)))
		})

		it("writes an nginx.conf that conditionally includes the Force HTTPS content", func() {
			err := generator.Generate(nginx.Configuration{
				NGINXConfLocation:   filepath.Join(workingDir, "nginx.conf"),
				WebServerForceHTTPS: true,
				WebServerRoot:       "./public",
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(filepath.Join(workingDir, "nginx.conf")).
				To(matchers.BeAFileMatching(ContainSubstring(`    root {{ env "APP_ROOT" }}/public;

    # If HTTP request is made, redirect to HTTPS requests
    set $updated_host $host;
    if ($http_x_forwarded_host != "") {
      set $updated_host $http_x_forwarded_host;
    }

    if ($http_x_forwarded_proto != "https") {
      return 301 https://$updated_host$request_uri;
    }
`)))
		})

		it("writes an nginx.conf that conditionally includes the stub_status module for basic status information", func() {
			err := generator.Generate(nginx.Configuration{
				NGINXConfLocation:   filepath.Join(workingDir, "nginx.conf"),
				NGINXStubStatusPort: "8083",
				WebServerRoot:       "./public",
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(filepath.Join(workingDir, "nginx.conf")).
				To(matchers.BeAFileMatching(ContainSubstring(`  # stub_status
  server {
    listen       8083;
    listen  [::]:8083;

    location /stub_status {
      stub_status;
    }
  }
`)))
		})

		it("writes an nginx.conf that conditionally includes the Basic Auth content", func() {
			err := generator.Generate(nginx.Configuration{
				NGINXConfLocation: filepath.Join(workingDir, "nginx.conf"),
				WebServerRoot:     "./public",
				BasicAuthFile:     "/some/file/path",
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(filepath.Join(workingDir, "nginx.conf")).
				To(matchers.BeAFileMatching(ContainSubstring(`    root {{ env "APP_ROOT" }}/public;

    # Require username + password authentication for access
    auth_basic "Password Protected";
    auth_basic_user_file /some/file/path;
`)))
		})

		context("failure cases", func() {
			context("destination file already exists and it's read-only", func() {
				it.Before(func() {
					Expect(os.WriteFile(filepath.Join(workingDir, "nginx.conf"), []byte("read-only file"), 0444)).To(Succeed())
				})
				it("returns an error", func() {
					err := generator.Generate(nginx.Configuration{
						NGINXConfLocation: filepath.Join(workingDir, "nginx.conf"),
					})
					Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("failed to create %[1]s: open %[1]s: permission denied", filepath.Join(workingDir, "nginx.conf")))))
				})
			})
		})
	})
}

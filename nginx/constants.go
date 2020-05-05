package nginx

const (
	NGINX      = "nginx"
	Dependency = NGINX // NOTE: alias for old constant name

	DepKey             = "dependency-sha"
	ConfigureBinKey    = "configure-bin-sha"
	ConfFile           = "nginx.conf"
	BuildpackYMLSource = "buildpack.yml"
)

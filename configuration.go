package main

import (
	"errors"
	"fmt"
	"github.com/containous/traefik/acme"
	"github.com/containous/traefik/provider"
	"github.com/containous/traefik/types"
	"regexp"
	"strings"
	"time"
)

// TraefikConfiguration holds GlobalConfiguration and other stuff
type TraefikConfiguration struct {
	GlobalConfiguration
	ConfigFile string `short:"c" description:"Timeout in seconds. Duration to give active requests a chance to finish during hot-reloads"`
}

// GlobalConfiguration holds global configuration (with providers, etc.).
// It's populated from the traefik configuration file passed as an argument to the binary.
type GlobalConfiguration struct {
	GraceTimeOut              int64                   `short:"g" description:"Configuration file to use (TOML)."`
	Debug                     bool                    `short:"d" description:"Enable debug mode"`
	AccessLogsFile            string                  `description:"Access logs file"`
	TraefikLogsFile           string                  `description:"Traefik logs file"`
	LogLevel                  string                  `short:"l" description:"Log level"`
	EntryPoints               EntryPoints             `description:"Entrypoints definition using format: --entryPoints='Name:http Address::8000 Redirect.EntryPoint:https' --entryPoints='Name:https Address::4442 TLS:tests/traefik.crt,tests/traefik.key'"`
	ACME                      *acme.ACME              `description:"Enable ACME (Let's Encrypt): automatic SSL"`
	DefaultEntryPoints        DefaultEntryPoints      `description:"Entrypoints to be used by frontends that do not specify any entrypoint"`
	ProvidersThrottleDuration time.Duration           `description:"Backends throttle duration: minimum duration between 2 events from providers before applying a new configuration. It avoids unnecessary reloads if multiples events are sent in a short amount of time."`
	MaxIdleConnsPerHost       int                     `description:"If non-zero, controls the maximum idle (keep-alive) to keep per-host.  If zero, DefaultMaxIdleConnsPerHost is used"`
	Retry                     *Retry                  `description:"Enable retry sending request if network error"`
	Docker                    *provider.Docker        `description:"Enable Docker backend"`
	File                      *provider.File          `description:"Enable File backend"`
	Web                       *WebProvider            `description:"Enable Web backend"`
	Marathon                  *provider.Marathon      `description:"Enable Marathon backend"`
	Consul                    *provider.Consul        `description:"Enable Consul backend"`
	ConsulCatalog             *provider.ConsulCatalog `description:"Enable Consul catalog backend"`
	Etcd                      *provider.Etcd          `description:"Enable Etcd backend"`
	Zookeeper                 *provider.Zookepper     `description:"Enable Zookeeper backend"`
	Boltdb                    *provider.BoltDb        `description:"Enable Boltdb backend"`
	Kubernetes                *provider.Kubernetes    `description:"Enable Kubernetes backend"`
}

// DefaultEntryPoints holds default entry points
type DefaultEntryPoints []string

// String is the method to format the flag's value, part of the flag.Value interface.
// The String method's output will be used in diagnostics.
func (dep *DefaultEntryPoints) String() string {
	return strings.Join(*dep, ",")
}

// Set is the method to set the flag value, part of the flag.Value interface.
// Set's argument is a string to be parsed to set the flag.
// It's a comma-separated list, so we split it.
func (dep *DefaultEntryPoints) Set(value string) error {
	entrypoints := strings.Split(value, ",")
	if len(entrypoints) == 0 {
		return errors.New("Bad DefaultEntryPoints format: " + value)
	}
	for _, entrypoint := range entrypoints {
		*dep = append(*dep, entrypoint)
	}
	return nil
}

// Get return the EntryPoints map
func (dep *DefaultEntryPoints) Get() interface{} { return DefaultEntryPoints(*dep) }

// SetValue sets the EntryPoints map with val
func (dep *DefaultEntryPoints) SetValue(val interface{}) {
	*dep = DefaultEntryPoints(val.(DefaultEntryPoints))
}

// Type is type of the struct
func (dep *DefaultEntryPoints) Type() string {
	return fmt.Sprint("defaultentrypoints")
}

// EntryPoints holds entry points configuration of the reverse proxy (ip, port, TLS...)
type EntryPoints map[string]*EntryPoint

// String is the method to format the flag's value, part of the flag.Value interface.
// The String method's output will be used in diagnostics.
func (ep *EntryPoints) String() string {
	return fmt.Sprintf("%+v", *ep)
}

// Set is the method to set the flag value, part of the flag.Value interface.
// Set's argument is a string to be parsed to set the flag.
// It's a comma-separated list, so we split it.
func (ep *EntryPoints) Set(value string) error {
	regex := regexp.MustCompile("(?:Name:(?P<Name>\\S*))\\s*(?:Address:(?P<Address>\\S*))?\\s*(?:TLS:(?P<TLS>\\S*))?\\s*(?:Redirect.EntryPoint:(?P<RedirectEntryPoint>\\S*))?\\s*(?:Redirect.Regex:(?P<RedirectRegex>\\S*))?\\s*(?:Redirect.Replacement:(?P<RedirectReplacement>\\S*))?")
	match := regex.FindAllStringSubmatch(value, -1)
	if match == nil {
		return errors.New("Bad EntryPoints format: " + value)
	}
	matchResult := match[0]
	result := make(map[string]string)
	for i, name := range regex.SubexpNames() {
		if i != 0 {
			result[name] = matchResult[i]
		}
	}
	var tls *TLS
	if len(result["TLS"]) > 0 {
		certs := Certificates{}
		if err := certs.Set(result["TLS"]); err != nil {
			return err
		}
		tls = &TLS{
			Certificates: certs,
		}
	}
	var redirect *Redirect
	if len(result["RedirectEntryPoint"]) > 0 || len(result["RedirectRegex"]) > 0 || len(result["RedirectReplacement"]) > 0 {
		redirect = &Redirect{
			EntryPoint:  result["RedirectEntryPoint"],
			Regex:       result["RedirectRegex"],
			Replacement: result["RedirectReplacement"],
		}
	}

	(*ep)[result["Name"]] = &EntryPoint{
		Address:  result["Address"],
		TLS:      tls,
		Redirect: redirect,
	}

	return nil
}

// Get return the EntryPoints map
func (ep *EntryPoints) Get() interface{} { return EntryPoints(*ep) }

// SetValue sets the EntryPoints map with val
func (ep *EntryPoints) SetValue(val interface{}) {
	*ep = EntryPoints(val.(EntryPoints))
}

// Type is type of the struct
func (ep *EntryPoints) Type() string {
	return fmt.Sprint("entrypoints²")
}

// EntryPoint holds an entry point configuration of the reverse proxy (ip, port, TLS...)
type EntryPoint struct {
	Network  string
	Address  string
	TLS      *TLS
	Redirect *Redirect
}

// Redirect configures a redirection of an entry point to another, or to an URL
type Redirect struct {
	EntryPoint  string
	Regex       string
	Replacement string
}

// TLS configures TLS for an entry point
type TLS struct {
	Certificates Certificates
}

// Certificates defines traefik certificates type
type Certificates []Certificate

// String is the method to format the flag's value, part of the flag.Value interface.
// The String method's output will be used in diagnostics.
func (certs *Certificates) String() string {
	if len(*certs) == 0 {
		return ""
	}
	return (*certs)[0].CertFile + "," + (*certs)[0].KeyFile
}

// Set is the method to set the flag value, part of the flag.Value interface.
// Set's argument is a string to be parsed to set the flag.
// It's a comma-separated list, so we split it.
func (certs *Certificates) Set(value string) error {
	files := strings.Split(value, ",")
	if len(files) != 2 {
		return errors.New("Bad certificates format: " + value)
	}
	*certs = append(*certs, Certificate{
		CertFile: files[0],
		KeyFile:  files[1],
	})
	return nil
}

// Type is type of the struct
func (certs *Certificates) Type() string {
	return fmt.Sprint("certificates")
}

// Certificate holds a SSL cert/key pair
type Certificate struct {
	CertFile string
	KeyFile  string
}

// Retry contains request retry config
type Retry struct {
	Attempts int   `description:"Number of attempts"`
	MaxMem   int64 `description:"Maximum request body to be stored in memory in Mo"`
}

// NewTraefikDefaultPointersConfiguration creates a TraefikConfiguration with pointers default values
func NewTraefikDefaultPointersConfiguration() *TraefikConfiguration {
	//default Docker
	var defaultDocker provider.Docker
	defaultDocker.Watch = true
	defaultDocker.Endpoint = "unix:///var/run/docker.sock"
	defaultDocker.TLS = &provider.DockerTLS{}

	// default File
	var defaultFile provider.File
	defaultFile.Watch = true
	defaultFile.Filename = "" //needs equivalent to  viper.ConfigFileUsed()

	// default Web
	var defaultWeb WebProvider
	defaultWeb.Address = ":8080"

	// default Marathon
	var defaultMarathon provider.Marathon
	defaultMarathon.Watch = true
	defaultMarathon.Endpoint = "http://127.0.0.1:8080"
	defaultMarathon.ExposedByDefault = true

	// default Consul
	var defaultConsul provider.Consul
	defaultConsul.Watch = true
	defaultConsul.Endpoint = "127.0.0.1:8500"
	defaultConsul.Prefix = "/traefik"
	defaultConsul.TLS = &provider.KvTLS{}

	// default ConsulCatalog
	var defaultConsulCatalog provider.ConsulCatalog
	defaultConsulCatalog.Endpoint = "127.0.0.1:8500"

	// default Etcd
	var defaultEtcd provider.Etcd
	defaultEtcd.Watch = true
	defaultEtcd.Endpoint = "127.0.0.1:400"
	defaultEtcd.Prefix = "/traefik"
	defaultEtcd.TLS = &provider.KvTLS{}

	//default Zookeeper
	var defaultZookeeper provider.Zookepper
	defaultZookeeper.Watch = true
	defaultZookeeper.Endpoint = "127.0.0.1:2181"
	defaultZookeeper.Prefix = "/traefik"

	//default Boltdb
	var defaultBoltDb provider.BoltDb
	defaultBoltDb.Watch = true
	defaultBoltDb.Endpoint = "127.0.0.1:4001"
	defaultBoltDb.Prefix = "/traefik"

	//default Kubernetes
	var defaultKubernetes provider.Kubernetes
	defaultKubernetes.Watch = true
	defaultKubernetes.Endpoint = "127.0.0.1:8080"

	defaultConfiguration := GlobalConfiguration{
		Docker:        &defaultDocker,
		File:          &defaultFile,
		Web:           &defaultWeb,
		Marathon:      &defaultMarathon,
		Consul:        &defaultConsul,
		ConsulCatalog: &defaultConsulCatalog,
		Etcd:          &defaultEtcd,
		Zookeeper:     &defaultZookeeper,
		Boltdb:        &defaultBoltDb,
		Kubernetes:    &defaultKubernetes,
		Retry:         &Retry{MaxMem: 2},
	}
	return &TraefikConfiguration{
		GlobalConfiguration: defaultConfiguration,
	}
}

// NewTraefikConfiguration creates a TraefikConfiguration with default values
func NewTraefikConfiguration() *TraefikConfiguration {
	return &TraefikConfiguration{
		GlobalConfiguration: GlobalConfiguration{
			GraceTimeOut:              10,
			AccessLogsFile:            "",
			TraefikLogsFile:           "",
			LogLevel:                  "ERROR",
			EntryPoints:               map[string]*EntryPoint{},
			DefaultEntryPoints:        []string{},
			ProvidersThrottleDuration: time.Duration(2 * time.Second),
			MaxIdleConnsPerHost:       200,
		},
		ConfigFile: "",
	}
}

type configs map[string]*types.Configuration

/*
Copyright The Voyager Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"encoding/json"

	core "k8s.io/api/core/v1"
)

const (
	EngressKey = "ingress.appscode.com"

	APISchema        = EngressKey + "/" + "api-schema" // APISchema = {APIGroup}/{APIVersion}
	APISchemaEngress = "voyager.appscode.com" + "/v1beta1"
	APISchemaIngress = "extensions/v1beta1"

	VoyagerPrefix = "voyager-"

	// LB stats options
	StatsOn          = EngressKey + "/" + "stats"
	StatsPort        = EngressKey + "/" + "stats-port"
	StatsSecret      = EngressKey + "/" + "stats-secret-name"
	StatsServiceName = EngressKey + "/" + "stats-service-name"
	DefaultStatsPort = 56789

	LBTypeHostPort     = "HostPort"
	LBTypeNodePort     = "NodePort"
	LBTypeLoadBalancer = "LoadBalancer" // default
	LBTypeInternal     = "Internal"
	LBType             = EngressKey + "/" + "type"

	// Replicas specify # of HAProxy pods run (default 1)
	Replicas = EngressKey + "/" + "replicas"

	// IP to be assigned to cloud load balancer
	LoadBalancerIP = EngressKey + "/" + "load-balancer-ip" // IP or empty

	// BackendWeight is the weight value of a Pod that was
	// addressed by the Endpoint, this weight will be added to server backend.
	// Traffic will be forwarded according to there weight.
	BackendWeight = EngressKey + "/" + "backend-weight"

	// https://github.com/appscode/voyager/issues/103
	// ServiceAnnotations is user provided annotations map that will be
	// applied to the service of that LoadBalancer.
	// ex: "ingress.appscode.com/annotations-service": {"key": "val"}
	ServiceAnnotations        = EngressKey + "/" + "annotations-service"
	LastAppliedAnnotationKeys = EngressKey + "/" + "last-applied-annotation-keys"

	// PodAnnotations is user provided annotations map that will be
	// applied to the Pods (Deployment/ DaemonSet) of that LoadBalancer.
	// ex: "ingress.appscode.com/annotations-pod": {"key": "val"}
	PodAnnotations = EngressKey + "/" + "annotations-pod"

	// Preserves source IP for LoadBalancer type ingresses. The actual configuration
	// generated depends on the underlying cloud provider.
	//
	// - gce, gke, azure: Sets "ExternalTrafficPolicy" to "Local" for HAProxy services.
	// Ref: https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/#preserving-the-client-source-ip
	//
	// - aws: Enables accept-proxy.
	// Enforces the use of the PROXY protocol over any connection accepted by any of
	// the sockets declared on the same line. Versions 1 and 2 of the PROXY protocol
	// are supported and correctly detected. The PROXY protocol dictates the layer
	// 3/4 addresses of the incoming connection to be used everywhere an address is
	// used, with the only exception of "tcp-request connection" rules which will
	// only see the real connection address. Logs will reflect the addresses
	// indicated in the protocol, unless it is violated, in which case the real
	// address will still be used.  This keyword combined with support from external
	// components can be used as an efficient and reliable alternative to the
	// X-Forwarded-For mechanism which is not always reliable and not even always
	// usable. See also "tcp-request connection expect-proxy" for a finer-grained
	// setting of which client is allowed to use the protocol.
	// Ref: https://github.com/kubernetes/kubernetes/blob/release-1.5/pkg/cloudprovider/providers/aws/aws.go#L79
	KeepSourceIP = EngressKey + "/" + "keep-source-ip"

	// https://github.com/appscode/voyager/issues/1128
	HealthCheckNodeport = EngressKey + "/" + "health-check-nodeport"

	// Enforces the use of the PROXY protocol over any connection accepted by HAProxy.
	AcceptProxy = EngressKey + "/" + "accept-proxy"

	// Enforces use of the PROXY protocol over any connection established to this server.
	// Possible values are "v1", "v2", "v2-ssl" and "v2-ssl-cn"
	SendProxy = EngressKey + "/" + "send-proxy"

	// Annotations applied to resources offshoot from an ingress
	OriginAPISchema = EngressKey + "/" + "origin-api-schema" // APISchema = {APIGroup}/{APIVersion}
	OriginName      = EngressKey + "/" + "origin-name"

	// https://github.com/appscode/voyager/issues/280
	// Supports all valid timeout option for defaults section of HAProxy
	// https://cbonte.github.io/haproxy-dconv/1.7/configuration.html#4.2-timeout%20check
	// expects a json encoded map
	// ie: "ingress.appscode.com/default-timeout": {"client": "5s"}
	//
	// If the annotation is not set default values used to config defaults section will be:
	//
	// timeout  connect         5s
	// timeout  client          50s
	// timeout  client-fin      50s
	// timeout  server          50s
	// timeout  tunnel          50s
	DefaultsTimeOut = EngressKey + "/" + "default-timeout"

	// Defines the maximum time allowed to perform a clean soft-stop.
	// https://cbonte.github.io/haproxy-dconv/1.9/configuration.html#hard-stop-after
	HardStopAfter        = EngressKey + "/" + "hard-stop-after"
	DefaultHardStopAfter = "30s"

	// https://github.com/appscode/voyager/issues/343
	// Supports all valid options for defaults section of HAProxy config
	// https://cbonte.github.io/haproxy-dconv/1.7/configuration.html#4.2-option%20abortonclose
	// from the list from here
	// expects a json encoded map
	// ie: "ingress.appscode.com/default-option": '{"http-keep-alive": "true", "dontlognull": "true", "clitcpka": "false"}'
	// This will be appended in the defaults section of HAProxy as
	//
	//   option http-keep-alive
	//   option dontlognull
	//   no option clitcpka
	//
	DefaultsOption = EngressKey + "/" + "default-option"

	// Available Options:
	//
	// ssl: https://cbonte.github.io/haproxy-dconv/1.8/configuration.html#5.2-ssl
	// 		Creates a TLS/SSL socket when connecting to this server in order to cipher/decipher the traffic.
	//   	If verify not set the following error may occurred:
	//    		[/etc/haproxy/haproxy.cfg:49] verify is enabled by default but no CA file specified.
	//    		If you're running on a LAN where you're certain to trust the server's certificate,
	//    		please set an explicit 'verify none' statement on the 'server' line, or use
	//    		'ssl-server-verify none' in the global section to disable server-side verifications by default.
	//
	// verify [none|required]: https://cbonte.github.io/haproxy-dconv/1.8/configuration.html#5.2-verify
	//   	Sets HAProxy‘s behavior regarding the certificated presented by the server:
	//     		- none : Doesn’t verify the certificate of the server
	//     		- required (default value) : TLS handshake is aborted if the validation of the certificate presented by the server returns an error.
	//
	// verifyhost <hostname>: https://cbonte.github.io/haproxy-dconv/1.8/configuration.html#5.2-verifyhost
	//    	Sets a <hostname> to look for in the Subject and SubjectAlternateNames fields provided in the
	//    	certificate sent by the server. If <hostname> can’t be found, then the TLS handshake is aborted.
	//    	This only applies when verify required is configured.
	//
	// Example: ingress.appscode.com/backend-tls: "ssl verify none"
	//
	// If this annotation is not set HAProxy will connect to backend as http.
	// This value should not be set if the backend do not support https resolution.
	BackendTLSOptions = EngressKey + "/backend-tls"

	// Specify a method to stick clients to origins across requests.
	// Only supported value is "cookie".
	// Annotations could  be applied to either Ingress or backend Service
	// If applied to Ingress, all the backend connections would be sticky
	// If applied to Service and Ingress do not have this annotation only
	// connection to that backend service will be sticky.
	IngressAffinity = EngressKey + "/affinity"
	// When affinity is set to "cookie", the name of the cookie to use.
	IngressAffinitySessionCookieName = EngressKey + "/session-cookie-name"
	// When affinity is set to "cookie", the hash algorithm used: md5, sha, index.
	IngressAffinitySessionCookieHash = EngressKey + "/session-cookie-hash"

	// Basic Auth: Follows ingress controller standard
	// https://github.com/kubernetes/ingress/tree/master/examples/auth/basic/haproxy
	// HAProxy Ingress read user and password from auth file stored on secrets, one
	// user and password per line.
	// Each line of the auth file should have:
	// user and insecure password separated with a pair of colons: <username>::<plain-text-passwd>; or
	// user and an encrypted password separated with colons: <username>:<encrypted-passwd>
	// Secret name, realm and type are configured with annotations in the ingress
	// Auth can only be applied to HTTP backends.
	// Only supported type is basic
	AuthType = EngressKey + "/auth-type"

	// an optional string with authentication realm
	AuthRealm = EngressKey + "/auth-realm"

	// name of the auth secret
	AuthSecret = EngressKey + "/auth-secret"

	// Name of secret for TLS client certification validation.
	AuthTLSSecret = EngressKey + "/auth-tls-secret"

	// The page that user should be redirected in case of Auth error
	AuthTLSErrorPage = EngressKey + "/auth-tls-error-page"

	// Enables verification of client certificates.
	AuthTLSVerifyClient = EngressKey + "/auth-tls-verify-client"

	// Enables CORS headers in response.
	// Setting this annotations in ingress will add CORS headers to all HTTP
	// frontend. If we need to add cors headers only on specific frontend we can also
	// configure this using FrontendRules for specific frontend.
	// http://blog.nasrulhazim.com/2017/07/haproxy-setting-up-cors/
	CORSEnabled          = EngressKey + "/enable-cors"
	CORSAllowedMethods   = EngressKey + "/cors-allow-methods"
	CORSAllowedHeaders   = EngressKey + "/cors-allow-headers"
	CORSAllowedOrigin    = EngressKey + "/cors-allow-origin"
	CORSAllowCredentials = EngressKey + "/cors-allow-credentials"

	// Maximum http request body size. This returns the advertised length of the HTTP request's body in bytes. It
	// will represent the advertised Content-Length header
	// http://cbonte.github.io/haproxy-dconv/1.7/configuration.html#7.3.6-req.body_size
	//
	ProxyBodySize = EngressKey + "/proxy-body-size"

	// Pass TLS connections directly to backend; do not offload.
	SSLPassthrough = EngressKey + "/ssl-passthrough"

	EnableHSTS = EngressKey + "/hsts"
	// This specifies the time (in seconds) the browser should connect to the server using the HTTPS connection.
	// https://blog.stackpath.com/glossary/hsts/
	HSTSMaxAge  = EngressKey + "/hsts-max-age"
	HSTSPreload = EngressKey + "/hsts-preload"
	// If specified, this HSTS rule applies to all of the site's subdomains as well.
	HSTSIncludeSubDomains = EngressKey + "/hsts-include-subdomains"

	WhitelistSourceRange = EngressKey + "/whitelist-source-range"
	MaxConnections       = EngressKey + "/max-connections"

	// https://github.com/appscode/voyager/issues/552
	UseNodePort      = EngressKey + "/use-node-port"
	SSLRedirect      = EngressKey + "/ssl-redirect"
	ForceSSLRedirect = EngressKey + "/force-ssl-redirect"

	// https://github.com/appscode/voyager/issues/525
	ErrorFiles = EngressKey + "/errorfiles"

	// Limit requests per second per IP address
	// http://cbonte.github.io/haproxy-dconv/1.8/configuration.html#7.3.3-sc_conn_rate
	// https://serverfault.com/a/679172/349346
	// https://discourse.haproxy.org/t/solved-how-to-configure-basic-ddos-protection-when-behind-aws-elb-x-forwarded-for/932
	// https://www.haproxy.com/blog/use-a-load-balancer-as-a-first-row-of-defense-against-ddos/
	LimitRPS = EngressKey + "/limit-rps"
	// Limit requests per minute per IP address
	LimitRPM = EngressKey + "/limit-rpm"

	// http://cbonte.github.io/haproxy-dconv/1.8/configuration.html#7.3.3-src_conn_cur
	// https://www.haproxy.com/blog/use-a-load-balancer-as-a-first-row-of-defense-against-ddos/
	LimitConnection = EngressKey + "/limit-connection"

	// https://github.com/appscode/voyager/issues/683
	// https://www.haproxy.com/documentation/aloha/7-0/haproxy/healthchecks/
	CheckHealth     = EngressKey + "/" + "check"
	CheckHealthPort = EngressKey + "/" + "check-port"

	// https://github.com/kubernetes/ingress-nginx/blob/master/docs/examples/rewrite/README.md
	RewriteTarget = EngressKey + "/" + "rewrite-target"

	// Workload controller to use run HAProxy pods
	WorkloadKind = EngressKey + "/" + "workload-kind"

	// ref: https://github.com/appscode/voyager/issues/1054
	NodeSelector = EngressKey + "/" + "node-selector"
	Tolerations  = EngressKey + "/" + "tolerations"

	// https://github.com/appscode/voyager/issues/1210
	// http://cbonte.github.io/haproxy-dconv/1.8/configuration.html#agent-check
	AgentPort     = EngressKey + "/" + "agent-port"
	AgentInterval = EngressKey + "/" + "agent-interval"
)

const (
	ACMEUserEmail        = "ACME_EMAIL"
	ACMEUserPrivatekey   = "ACME_USER_PRIVATE_KEY"
	ACMERegistrationData = "ACME_REGISTRATION_DATA"
	ACMEServerURL        = "ACME_SERVER_URL"
)

type ProxyProtocolVersion string

const (
	proxyProtocolV1      ProxyProtocolVersion = "v1"
	proxyProtocolV2      ProxyProtocolVersion = "v2"
	proxyProtocolV2SSL   ProxyProtocolVersion = "v2-ssl"
	proxyProtocolV2SSLCN ProxyProtocolVersion = "v2-ssl-cn"
)

func ProxyProtocolCommand(version string) string {
	switch ProxyProtocolVersion(version) {
	case proxyProtocolV1:
		return "send-proxy"
	case proxyProtocolV2, proxyProtocolV2SSL, proxyProtocolV2SSLCN:
		return "send-proxy-" + version
	default:
		return ""
	}
}

func (r Ingress) OffshootName() string {
	return VoyagerPrefix + r.Name
}

const (
	// ref: https://github.com/kubernetes/ingress-nginx/blob/master/docs/annotations.md#cors-related
	CORSDefaultAllowedMethods = "GET,PUT,POST,DELETE,PATCH,OPTIONS"
	CORSDefaultAllowedHeaders = "DNT,X-CustomHeader,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Authorization"
)

func getTolerations(m map[string]string, key string) (interface{}, error) {
	if m == nil {
		return nil, nil
	}
	v, ok := m[key]
	if !ok {
		return nil, nil
	}

	var out []core.Toleration
	err := json.Unmarshal([]byte(v), &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

var timeoutKeys = []string{
	"check",
	"client",
	"client-fin",
	"connect",
	"http-keep-alive",
	"http-request",
	"queue",
	"server",
	"server-fin",
	"tarpit",
	"tunnel",
}

var timeoutDefaults = map[string]string{
	// Maximum time to wait for a connection attempt to a server to succeed.
	"connect": "5s",

	// Maximum inactivity time on the client side.
	// Applies when the client is expected to acknowledge or send data.
	"client": "50s",

	// Inactivity timeout on the client side for half-closed connections.
	// Applies when the client is expected to acknowledge or send data
	// while one direction is already shut down.
	"client-fin": "50s",

	// Maximum inactivity time on the server side.
	"server": "50s",

	// Timeout to use with WebSocket and CONNECT
	"tunnel": "50s",
}

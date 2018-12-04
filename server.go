package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-ocf/go-coap"
	"github.com/kelseyhightower/envconfig"
	"github.com/valyala/fasthttp"
)

//config for application
type config struct {
	KeepaliveTime     time.Duration `envconfig:"KEEPALIVE_TIME" default:"3600s"`
	KeepaliveInterval time.Duration `envconfig:"KEEPALIVE_INTERVAL" default:"5s"`
	KeepaliveRetry    int           `envconfig:"KEEPALIVE_RETRY" default:"5"`
	Addr              string        `envconfig:"ADDRESS" default:"0.0.0.0:5684"`
	Net               string        `envconfig:"NETWORK" default:"tcp"`
	AuthHost          string        `envconfig:"AUTH_HOST"  default:"127.0.0.1"`
	AuthProtocol      authProto     `envconfig:"AUTH_PROTOCOL"  default:"http"`
}

//config for application
type tlsConfig struct {
	Certificate    string `envconfig:"TLS_CERTIFICATE" required:"true"`
	CertificateKey string `envconfig:"TLS_CERTIFICATE_KEY" required:"true"`
	CAPool         string `envconfig:"TLS_CA_POOL" required:"true"`
}

//Server a configuration of coapgateway
type Server struct {
	Addr              string        // Address to listen on, ":COAP" if empty.
	Net               string        // if "tcp" or "tcp-tls" (COAP over TLS) it will invoke a TCP listener, otherwise an UDP one
	TLSConfig         *tls.Config   // TLS connection configuration
	keepaliveTime     time.Duration // the duration in seconds between two keepalive transmissions in idle condition. TCP keepalive period is required to be configurable and by default is set to 1 hour.
	keepaliveInterval time.Duration // the duration in seconds between two successive keepalive retransmissions, if acknowledgement to the previous keepalive transmission is not received.
	keepaliveRetry    int           // the number of retransmissions to be carried out before declaring that remote end is not available.
	AuthHost          string        // IP/DOMAIN where gateway will create connections for authentification
	AuthProtocol      string        // http or https

	clientContainer *ClientContainer
	httpClient      *fasthttp.Client
}

func setupTLS() (*tls.Config, error) {
	cfg := &tlsConfig{}
	if err := envconfig.Process(os.Args[0], cfg); err != nil {
		return nil, err
	}

	cert, err := tls.LoadX509KeyPair(cfg.Certificate, cfg.CertificateKey)
	if err != nil {
		return nil, err
	}

	caRootPool := x509.NewCertPool()
	caIntermediatesPool := x509.NewCertPool()

	err = filepath.Walk(cfg.CAPool, func(path string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}

		// check if it is a regular file (not dir)
		if info.Mode().IsRegular() {
			certPEMBlock, err := ioutil.ReadFile(path)
			if err != nil {
				log.Errorf("Cannot read file '%v': %v", path, err)
				return nil
			}
			certDERBlock, _ := pem.Decode(certPEMBlock)
			if certDERBlock == nil {
				log.Errorf("Cannot decode der block '%v'", path)
				return nil
			}
			if certDERBlock.Type != "CERTIFICATE" {
				log.Errorf("DER block is not certificate '%v'", path)
				return nil
			}
			caCert, err := x509.ParseCertificate(certDERBlock.Bytes)
			if err != nil {
				log.Errorf("Cannot parse certificate '%v': %v", path, err)
				return nil
			}
			if bytes.Compare(caCert.RawIssuer, caCert.RawSubject) == 0 && caCert.IsCA {
				log.Infof("Adding root certificate '%v'", path)
				caRootPool.AddCert(caCert)
			} else if caCert.IsCA {
				log.Infof("Adding intermediate certificate '%v'", path)
				caIntermediatesPool.AddCert(caCert)
			} else {
				log.Warnf("Ignoring certificate '%v'", path)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	if len(caRootPool.Subjects()) == 0 {
		return nil, ErrEmptyCARootPool
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAnyClientCert,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifyChains [][]*x509.Certificate) error {
			for _, rawCert := range rawCerts {
				cert, err := x509.ParseCertificates(rawCert)
				if err != nil {
					return err
				}
				//TODO verify revocation
				for _, c := range cert {
					_, err := c.Verify(x509.VerifyOptions{
						Intermediates: caIntermediatesPool,
						Roots:         caRootPool,
						CurrentTime:   time.Now(),
						KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
					})
					if err != nil {
						return err
					}
				}
				//TODO verify EKU - need to use ASN decoding
			}
			return nil
		},
	}, nil
}

type authProto string

func (a *authProto) Decode(value string) error {
	switch value {
	case "http", "https":
		*a = authProto(value)
		return nil
	default:
		return fmt.Errorf("Unsupported protocol type %v", value)
	}
}

//NewServer setup coap gateway
func NewServer() (*Server, error) {
	var cfg config
	if err := envconfig.Process(os.Args[0], &cfg); err != nil {
		return nil, err
	}

	s := Server{
		keepaliveTime:     cfg.KeepaliveTime,
		keepaliveInterval: cfg.KeepaliveInterval,
		keepaliveRetry:    cfg.KeepaliveRetry,
		Net:               cfg.Net,
		Addr:              cfg.Addr,
		AuthHost:          cfg.AuthHost,
		AuthProtocol:      string(cfg.AuthProtocol),

		clientContainer: &ClientContainer{sessions: make(map[string]*Session)},
		httpClient:      &fasthttp.Client{},
	}

	if strings.Contains(s.Net, "tls") {
		var err error
		s.TLSConfig, err = setupTLS()
		if err != nil {
			return nil, err
		}
	}

	return &s, nil
}

func validateCommandCode(s coap.ResponseWriter, req *coap.Request, server *Server, fnc func(s coap.ResponseWriter, req *coap.Request, server *Server)) {
	decodeMsgToDebug(req.Msg, "MESSAGE_FROM_CLIENT")
	switch req.Msg.Code() {
	case coap.POST, coap.DELETE, coap.PUT, coap.GET:
		fnc(s, req, server)
	case coap.Content:
		log.Infof("Unpaired message received from %v", req.Client.RemoteAddr())
	default:
		log.Errorf("Invalid code received %v from %v", req.Msg.Code(), req.Client.RemoteAddr())
	}
}

//NewCoapServer setup coap server
func (server *Server) NewCoapServer() *coap.Server {
	mux := coap.NewServeMux()
	mux.DefaultHandle(coap.HandlerFunc(func(s coap.ResponseWriter, req *coap.Request) {
		validateCommandCode(s, req, server, defaultHandler)
	}))
	mux.Handle(oicRd, coap.HandlerFunc(func(s coap.ResponseWriter, req *coap.Request) {
		validateCommandCode(s, req, server, oicRdHandler)
	}))
	mux.Handle(oicSecAccount, coap.HandlerFunc(func(s coap.ResponseWriter, req *coap.Request) {
		validateCommandCode(s, req, server, oicSecAccountHandler)
	}))
	mux.Handle(oicSecSession, coap.HandlerFunc(func(s coap.ResponseWriter, req *coap.Request) {
		validateCommandCode(s, req, server, oicSecSessionHandler)
	}))

	return &coap.Server{
		Net:       server.Net,
		Addr:      server.Addr,
		TLSConfig: server.TLSConfig,
		Handler:   mux,
		NotifySessionNewFunc: func(s *coap.ClientCommander) {
			server.clientContainer.add(server, s)
		},
		NotifySessionEndFunc: func(s *coap.ClientCommander, err error) {
			server.clientContainer.remove(s)
		},
	}
}

//ListenAndServe starts a coapgateway on the configured address in *Server.
func (server *Server) ListenAndServe() error {
	return server.NewCoapServer().ListenAndServe()
}

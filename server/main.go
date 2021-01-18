package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"sigs.k8s.io/yaml"

	"k8s.io/apimachinery/pkg/util/proxy"

	"github.com/gorilla/mux"
	"github.com/rancher/remotedialer"
	"github.com/sirupsen/logrus"
)

var (
	transports = map[string]*http.Transport{}
	l          sync.Mutex
	counter    int64
)

type PeerConfig struct {
	URL   string `yaml:"url"`
	ID    string `yaml:"id"`
	Token string `yaml:"token"`
}

type PeersConfig struct {
	Peers []PeerConfig `yaml:"peers"`
}

func authorizer(req *http.Request) (string, bool, error) {
	id := req.Header.Get("x-tunnel-id")
	return id, id != "", nil
}

func Client(server *remotedialer.Server, rw http.ResponseWriter, req *http.Request) {
	timeout := req.URL.Query().Get("timeout")
	if timeout == "" {
		timeout = "15"
	}

	vars := mux.Vars(req)
	clientKey := vars["id"]
	//url := fmt.Sprintf("%s://%s%s", vars["scheme"], vars["host"], vars["path"])
	transport := getTransport(server, clientKey, vars["host"], timeout)

	logrus.Printf("[%s] proxy %s %s %s", clientKey, req.Host, req.Method, req.URL.String())

	u := *req.URL
	u.Host = vars["host"]
	u.Path = vars["path"]
	u.Scheme = vars["scheme"]

	httpProxy := proxy.NewUpgradeAwareHandler(&u, transport, false, false, nil)
	httpProxy.ServeHTTP(rw, req)

}

func getTransport(server *remotedialer.Server, clientKey, targetHost, timeout string) *http.Transport {
	l.Lock()
	defer l.Unlock()

	key := fmt.Sprintf("%s/%s/%s", clientKey, targetHost, timeout)
	t := transports[key]
	if t != nil {
		return t
	}

	t = &http.Transport{
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return server.Dialer(clientKey, 15*time.Second)(network, targetHost)
		},
		TLSClientConfig: &tls.Config{
			// Ignore tls verify for now
			InsecureSkipVerify: true,
		},
	}

	transports[key] = t
	return t
}

func main() {
	var (
		addr            string
		peerID          string
		peerToken       string
		peers           string
		peersConfigFile string
		debug           bool
	)
	flag.StringVar(&addr, "listen", ":8123", "Listen address")
	flag.StringVar(&peerID, "id", "", "Peer ID")
	flag.StringVar(&peerToken, "token", "", "Peer Token")
	flag.StringVar(&peers, "peers", "", "Peers format id:token:url,id:token:url")
	flag.StringVar(&peersConfigFile, "peers-config-file", "", "Peers config file")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.Parse()

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
		remotedialer.PrintTunnelData = true
	}

	fmt.Printf(`
Debug Flags:
  listen: %s
  id: %s
  token: %s
  peers: %s
  peersConfigFile: %s
`, addr, peerID, peerToken, peers, peersConfigFile)
	handler := remotedialer.New(authorizer, remotedialer.DefaultErrorWriter)
	handler.PeerToken = peerToken
	handler.PeerID = peerID

	if peersConfigFile != "" {
		b, err := ioutil.ReadFile(peersConfigFile)
		if err != nil {
			panic("failed to read peers config file: " + err.Error())
		}
		pc := &PeersConfig{}
		err = yaml.Unmarshal(b, pc)
		if err != nil {
			panic("failed to parse peers config file: " + err.Error())
		}
		for _, p := range pc.Peers {
			if p.ID == peerID {
				// Do not add myself as a peer
				continue
			}
			fmt.Printf("Adding peer %s @ %s with token %s...\n", p.ID, p.URL, p.Token)
			handler.AddPeer(p.URL, p.ID, p.Token)
		}

	}

	if peers != "" {
		for _, peer := range strings.Split(peers, ",") {
			parts := strings.SplitN(strings.TrimSpace(peer), ":", 3)
			if len(parts) != 3 {
				continue
			}
			handler.AddPeer(parts[2], parts[0], parts[1])
		}
	}

	router := mux.NewRouter()
	router.Handle("/connect", handler)
	router.HandleFunc("/client/{id}/{scheme}/{host}{path:.*}", func(rw http.ResponseWriter, req *http.Request) {
		Client(handler, rw, req)
	})

	fmt.Println("Listening on ", addr)
	http.ListenAndServe(addr, router)
}

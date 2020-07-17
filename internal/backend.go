package internal

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"rdb/pkg"
	"rdb/pkg/usecases"
)

const (
	BridgeDescriptorReloadInterval = time.Minute * 5

	MinTransportWords = 3
	TransportPrefix   = "transport"
	ExtraInfoPrefix   = "extra-info"
)

var bridges *usecases.Bridges

func InitBackend(cfg *Config) {

	log.Println("Initialising backend.")

	mux := http.NewServeMux()
	endpoint := "/" + cfg.BackendEndpoint
	mux.Handle(endpoint, http.HandlerFunc(ResourceRequestHandler))

	go reloadBridgeDescriptors(cfg.ExtrainfoFile)

	if cfg.BackendCertfile != "" && cfg.BackendKeyfile != "" {
		log.Fatal(http.ListenAndServeTLS(cfg.BackendAddress, cfg.BackendCertfile, cfg.BackendKeyfile, mux))
	} else {
		log.Fatal(http.ListenAndServe(cfg.BackendAddress, mux))
	}
}

func reloadBridgeDescriptors(extrainfoFile string) {

	ticker := time.NewTicker(BridgeDescriptorReloadInterval)
	defer ticker.Stop()

	var err error
	for ; true; <-ticker.C {
		log.Println("Reloading bridge descriptors.")

		bridges, err = loadBridgesFromExtrainfo(extrainfoFile)
		if err != nil {
			log.Printf("Failed to reload bridge descriptors: %s", err)
		} else {
			log.Printf("Successfully reloaded %d bridge descriptors.", len(bridges.Bridges))
		}
	}
}

func ResourceRequestHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodPost {
		log.Println("POST request.")
	} else if r.Method == http.MethodGet {
		log.Println("GET request.")

		reqOrigin := r.URL.Query().Get("request_origin")
		resType := r.URL.Query().Get("resource_type")

		// reqOrigin := r.Form.Get("request_origin")
		// resType := r.Form.Get("resource_type")
		log.Printf("Request origin is %s; resource type is %s", reqOrigin, resType)
		var bs []*pkg.Transport
		log.Printf("Have %d bridges.", len(bridges.Bridges))
		i := 0
		for fingerprint, bridge := range bridges.Bridges {
			log.Printf("Selecting bridge %s", fingerprint)
			if len(bridge.Transports) == 0 {
				continue
			}
			bs = append(bs, bridge.Transports[0])
			i++
			if i == 5 {
				break
			}
		}
		jsonBlurb, err := json.Marshal(bs)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("Ran into error: %s", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, string(jsonBlurb))

	} else {
		log.Printf("Got unsupported HTTP %s request.", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// loadBridgesFromExtrainfo loads and returns bridges from Serge's extrainfo
// files.
func loadBridgesFromExtrainfo(extrainfoFile string) (*pkg.Bridges, error) {

	file, err := os.Open(extrainfoFile)
	if err != nil {
		log.Printf("Failed to open extrainfo file: %s", err)
		return nil, err
	}
	defer file.Close()

	extra, err := ParseExtrainfoDoc(file)
	if err != nil {
		log.Printf("Failed to read bridges from extrainfo file: %s", err)
		return nil, err
	}

	return extra, nil
}

// ParseExtrainfoDoc parses the given extra-info document and returns the
// content as a Bridges object.  Note that the extra-info document format is as
// it's produced by the bridge authority.
func ParseExtrainfoDoc(r io.Reader) (*pkg.Bridges, error) {

	var bridges = pkg.NewBridges()
	var b *pkg.Bridge

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		// We're dealing with a new extra-info block, i.e., a new bridge.
		if strings.HasPrefix(line, ExtraInfoPrefix) {
			b = pkg.NewBridge()
			words := strings.Split(line, " ")
			if len(words) != 3 {
				return nil, errors.New("incorrect number of words in 'extra-info' line")
			}
			b.Fingerprint = words[2]
			bridges.Bridges[b.Fingerprint] = b
		}
		// We're dealing with a bridge's transport protocols.  There may be
		// several.
		if strings.HasPrefix(line, TransportPrefix) {
			t := pkg.NewTransport()
			t.Fingerprint = b.Fingerprint
			err := populateTransportInfo(line, t)
			if err != nil {
				return nil, err
			}
			b.AddTransport(t)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return bridges, nil
}

// populateTransportInfo parses the given transport line of the format:
//   "transport" transportname address:port [arglist] NL
// ...and writes it to the given transport object.  See the specification for
// more details on what transport lines look like:
// <https://gitweb.torproject.org/torspec.git/tree/dir-spec.txt?id=2b31c63891a63cc2cad0f0710a45989071b84114#n1234>
func populateTransportInfo(transport string, t *pkg.Transport) error {

	if !strings.HasPrefix(transport, TransportPrefix) {
		return errors.New("no 'transport' prefix")
	}

	words := strings.Split(transport, " ")
	if len(words) < MinTransportWords {
		return errors.New("not enough arguments in 'transport' line")
	}
	t.Type = words[1]

	host, port, err := net.SplitHostPort(words[2])
	if err != nil {
		return err
	}
	addr, err := net.ResolveIPAddr("", host)
	if err != nil {
		return err
	}
	t.Address = pkg.IPAddr{net.IPAddr{addr.IP, addr.Zone}}
	p, err := strconv.Atoi(port)
	if err != nil {
		return err
	}
	t.Port = uint16(p)

	// We may be dealing with one or more key=value pairs.
	if len(words) > MinTransportWords {
		args := strings.Split(words[3], ",")
		for _, arg := range args {
			kv := strings.Split(arg, "=")
			if len(kv) != 2 {
				return fmt.Errorf("key:value pair in %q not separated by a '='", words[3])
			}
			t.Parameters[kv[0]] = kv[1]
		}
	}

	return nil
}

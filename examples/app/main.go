package main

import (
	"crypto"
	"crypto/aes"
	"crypto/dsa"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/md5"
	"crypto/rand"
	"crypto/rc4"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"reflect"
	"time"
	"unsafe"

	"example.com/example/app/internal/security"
	"golang.org/x/sys/unix"
	"gopkg.in/yaml.v3"
)

type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type KubeConfig struct {
	APIVersion string                 `yaml:"apiVersion" validate:"required"`
	Kind       string                 `yaml:"kind" validate:"required"`
	Metadata   Metadata               `yaml:"metadata"`
	Spec       map[string]interface{} `yaml:"spec"`
}

type Metadata struct {
	Name      string            `yaml:"name" validate:"required"`
	Namespace string            `yaml:"namespace"`
	Labels    map[string]string `yaml:"labels"`
}

func (k *KubeConfig) Validate() error {
	if k.APIVersion == "" {
		return fmt.Errorf("apiVersion is required")
	}
	if k.Kind == "" {
		return fmt.Errorf("kind is required")
	}
	if k.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}
	return nil
}

func (p *Person) GetName() string {
	return p.Name
}

func (p *Person) SetAge(age int) {
	p.Age = age
}

// DynamicMethodCaller demonstrates the call graph gap:
// This function -> reflect operations -> actual struct method
// The logs should show how to connect these dots using pointer addresses
func DynamicMethodCaller(obj *Person, methodName string, args []interface{}) []reflect.Value {
	val := reflect.ValueOf(obj)
	method := val.MethodByName(methodName)
	if !method.IsValid() {
		return nil
	}

	callArgs := make([]reflect.Value, len(args))
	for i, arg := range args {
		callArgs[i] = reflect.ValueOf(arg)
	}

	return method.Call(callArgs)
}

func main() {
	if !security.ValidateInput("test") {
		panic("validation failed")
	}

	for i := 0; i < 5; i++ {
		reflect.ValueOf(fmt.Sprintf("hello-%d", i))
	}

	people := []Person{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
	}

	data, _ := json.Marshal(people)

	var decoded []Person
	_ = json.Unmarshal(data, &decoded)

	yamlConfig := `
apiVersion: v1
kind: Deployment
metadata:
  name: example-app
  namespace: default
  labels:
    app: example
    version: "1.0"
spec:
  replicas: 3
  selector:
    matchLabels:
      app: example
`

	var config KubeConfig
	_ = yaml.Unmarshal([]byte(yamlConfig), &config)

	val := reflect.ValueOf(&config)
	method := val.MethodByName("Validate")
	if method.IsValid() {
		method.Call([]reflect.Value{})
	}

	// Test call graph reconstruction with pointer tracking
	person := &Person{Name: "Charlie", Age: 35}

	// Get method via reflection
	val = reflect.ValueOf(person)
	method = val.MethodByName("GetName")
	if !method.IsValid() {
		panic("method not found")
	}

	// Temporal gap: do some unsafe operations before calling
	unsafeArr := [16]int{3: 3, 9: 9, 11: 11}
	eleSize := int(unsafe.Sizeof(unsafeArr[0]))
	p9 := &unsafeArr[9]
	up9 := unsafe.Pointer(p9)
	_ = (*int)(unsafe.Add(up9, -6*eleSize))

	// Now call the method - internal ptr should still match
	name := method.Call([]reflect.Value{})
	if len(name) > 0 {
		fmt.Printf("Got name via reflection: %s\n", name[0].String())
	}

	// Another test with SetAge - also with temporal gap
	method2 := val.MethodByName("SetAge")
	if !method2.IsValid() {
		panic("method not found")
	}

	// More temporal operations
	anotherArr := [8]int{1: 1, 5: 5}
	p5 := &anotherArr[5]
	up5 := unsafe.Pointer(p5)
	_ = (*int)(unsafe.Add(up5, -3*int(unsafe.Sizeof(anotherArr[0]))))

	// Call with gap between MethodByName and Call
	method2.Call([]reflect.Value{reflect.ValueOf(40)})

	security.UnsafeMemoryOperator(5)

	// Test golang.org/x/sys/unix to see if it causes issues
	// This package uses unsafe but in ways that might not match our wrappers
	_ = unix.Getpid()

	demoCrypto()
}

func demoCrypto() {
	md5.Sum([]byte("test data for weak hash detection"))

	sha1.Sum([]byte("test data for deprecated hash"))

	sha256.Sum256([]byte("test data for approved hash"))

	_, _ = rc4.NewCipher([]byte("weak-rc4-key"))

	var dsaParams dsa.Parameters
	_ = dsa.GenerateParameters(&dsaParams, rand.Reader, dsa.L1024N160)
	var dsaPrivKey dsa.PrivateKey
	dsaPrivKey.Parameters = dsaParams
	_ = dsa.GenerateKey(&dsaPrivKey, rand.Reader)
	dsaR, dsaS, _ := dsa.Sign(rand.Reader, &dsaPrivKey, []byte("test-hash-data"))
	_ = dsa.Verify(&dsaPrivKey.PublicKey, []byte("test-hash-data"), dsaR, dsaS)

	key := make([]byte, 32)
	_, _ = rand.Read(key)
	_, _ = aes.NewCipher(key)

	crypto.SHA256.New()

	fmt.Println("Crypto operations completed")

	demoTLS()
}

func demoTLS() {
	cert, certPEM, _ := generateSelfSignedCert()

	serverConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP384,
		},
	}

	listener, err := tls.Listen("tcp", "127.0.0.1:0", serverConfig)
	if err != nil {
		fmt.Printf("TLS Listen error: %v\n", err)
		return
	}
	defer listener.Close()

	addr := listener.Addr().String()
	fmt.Printf("TLS server listening on %s\n", addr)

	done := make(chan struct{})
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		tlsConn := conn.(*tls.Conn)
		_ = tlsConn.Handshake()

		state := tlsConn.ConnectionState()
		fmt.Printf("Server: TLS %s, cipher 0x%04x\n",
			tlsVersionName(state.Version),
			state.CipherSuite)
		close(done)
	}()

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(certPEM)

	clientConfig := &tls.Config{
		RootCAs:    certPool,
		ServerName: "localhost",
		MinVersion: tls.VersionTLS12,
	}

	conn, err := tls.Dial("tcp", addr, clientConfig)
	if err != nil {
		fmt.Printf("TLS Dial error: %v\n", err)
		return
	}
	defer conn.Close()

	state := conn.ConnectionState()
	fmt.Printf("Client: TLS %s, cipher 0x%04x\n",
		tlsVersionName(state.Version),
		state.CipherSuite)

	<-done
	fmt.Println("TLS demo completed")
}

func generateSelfSignedCert() (tls.Certificate, []byte, []byte) {
	priv, _ := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Example Corp"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost"},
	}

	certDER, _ := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, _ := x509.MarshalECPrivateKey(priv)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	cert, _ := tls.X509KeyPair(certPEM, keyPEM)
	return cert, certPEM, keyPEM
}

func tlsVersionName(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("unknown (0x%04x)", version)
	}
}

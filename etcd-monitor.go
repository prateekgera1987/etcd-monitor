package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

var client *http.Client
var cw *cloudwatch.CloudWatch
var etcdName *string
var address *string
var awsRegion *string
var namespace *string
var signalCh chan os.Signal

type Health struct {
	IsHealthy bool `json:"health,string"`
}

func main() {
	signalCh = make(chan os.Signal, 1)

	defaultInterval := 60
	if i := os.Getenv("CHECK_INTERVAL"); i != "" {
		defaultIntervalEnv, err := strconv.Atoi(i)
		if err != nil {
			log.Fatal(err)
		}
		defaultInterval = defaultIntervalEnv
	}
	interval := flag.Int("interval", defaultInterval,
		"Time interval of how often to run the check (in seconds). "+
			"Overrides the CHECK_INTERVAL environment variable if set.")

	defaultAddress := "https://127.0.0.1:2379"
	if a := os.Getenv("ETCD_ADVERTISE_CLIENT_URLS"); a != "" {
		defaultAddress = a
	}
	address = flag.String("address", defaultAddress,
		"The address of the etcd server. "+
			"Overrides the ETCD_ADVERTISE_CLIENT_URLS environment variable if set.")

	defaultCaFile := ""
	if caEnv := os.Getenv("ETCDMON_CA_FILE"); caEnv != "" {
		defaultCaFile = caEnv
	}
	caFile := flag.String("ca-file", defaultCaFile, "A PEM eoncoded CA's certificate file.")

	defaultCertFile := ""
	if certEnv := os.Getenv("ETCDMON_CERT_FILE"); certEnv != "" {
		defaultCertFile = certEnv
	}
	certFile := flag.String("cert-file", defaultCertFile, "A PEM eoncoded certificate file.")

	defaultKeyFile := ""
	if keyEnv := os.Getenv("ETCDMON_KEY_FILE"); keyEnv != "" {
		defaultKeyFile = keyEnv
	}
	keyFile := flag.String("key-file", defaultKeyFile, "A PEM encoded private key file.")

	defaultEtcdName := "etcd"
	if n := os.Getenv("ETCD_NAME"); n != "" {
		defaultEtcdName = n
	}
	etcdName = flag.String("name", defaultEtcdName,
		"The name of the etcd cluster. This value will be used as CloudWatch dimension value. "+
			"Overrides the ETCD_NAME environment variable if set.")

	defaultNamespace := "etcd"
	if r := os.Getenv("METRIC_NAMESPACE"); r != "" {
		defaultNamespace = r
	}
	namespace = flag.String("namespace", defaultNamespace,
		"AWS CloudWatch metric namespace. "+
			"Overrides the METRIC_NAMESPACE environment variable if set.")

	defaultRegion := "us-east-1"
	if r := os.Getenv("AWS_REGION"); r != "" {
		defaultRegion = r
	}
	awsRegion = flag.String("region", defaultRegion,
		"AWS CloudWatch region. "+
			"Overrides the AWS_REGION environment variable if set.")

	flag.Parse()

	// Load client cert
	cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
	if err != nil {
		log.Fatal(err)
	}

	// Load CA cert
	caCert, err := ioutil.ReadFile(*caFile)
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}

	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	client = &http.Client{
		Transport: tr,
		Timeout:   time.Second * 5,
	}

	awsSession := session.New()
	awsSession.Config.WithRegion(*awsRegion)
	cw = cloudwatch.New(awsSession)

	fmt.Println("==> etcd Monitor Configuration:")
	fmt.Println("")
	fmt.Printf("\t      Check interval: %d (seconds)\n", *interval)
	fmt.Printf("\t        etcd Address: %s\n", *address)
	fmt.Printf("\t           etcd Name: %s\n", *etcdName)
	fmt.Printf("\tCloudWatch Namespace: %s\n", *namespace)
	fmt.Printf("\t          AWS Region: %s\n", *awsRegion)
	fmt.Println("")

	checkEtcdHealth()

	ticker := time.NewTicker(time.Duration(*interval) * time.Second)

	signal.Notify(signalCh)

	for {
		select {
		case <-ticker.C:
			checkEtcdHealth()

		case s := <-signalCh:
			log.Printf("[DEBUG] receiving signal: %q", s)
			ticker.Stop()
			os.Exit(0)
			return
		}
	}

}

func checkEtcdHealth() {
	resp, err := client.Get(fmt.Sprintf("%s/health", *address))
	if err != nil {
		log.Printf("[ERROR] Failed to connect to etcd: %s", err)
		reportUnhealtyCount(1.0)
		return
	}
	defer resp.Body.Close()

	buff, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[ERROR] Failed to get etcd health: %s", err)
		reportUnhealtyCount(1.0)
		return
	}

	var status Health
	err = json.Unmarshal(buff, &status)
	if err != nil {
		log.Printf("[ERROR] Invalid health response payload: %s", err)
		reportUnhealtyCount(1.0)
		return
	}

	if status.IsHealthy {
		reportUnhealtyCount(0.0)
	} else {
		reportUnhealtyCount(1.0)
	}
}

func reportUnhealtyCount(count float64) {
	if count > 0 {
		log.Printf("[INFO] etcd IS NOT healthy")
	} else {
		log.Printf("[INFO] etcd is healthy")
	}

	params := &cloudwatch.PutMetricDataInput{
		MetricData: []*cloudwatch.MetricDatum{
			{
				MetricName: aws.String("UnhealthyCount"),
				Dimensions: []*cloudwatch.Dimension{
					{
						Name:  aws.String("By cluster"),
						Value: aws.String(*etcdName),
					},
				},
				StatisticValues: &cloudwatch.StatisticSet{
					Maximum:     aws.Float64(count),
					Minimum:     aws.Float64(count),
					SampleCount: aws.Float64(1.0),
					Sum:         aws.Float64(count),
				},
				Timestamp: aws.Time(time.Now()),
				Unit:      aws.String("Count"),
			},
		},
		Namespace: aws.String(*namespace),
	}

	_, err := cw.PutMetricData(params)
	if err != nil {
		log.Println(err.Error())
		return
	}
}

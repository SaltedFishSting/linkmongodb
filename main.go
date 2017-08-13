package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/push"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/yaml.v2"
)

//配置文件yaml
type RConfig struct {
	Gw struct {
		Addr           string `yaml:"addr"`
		HttpListenPort int    `yaml:"httpListenPort"`
		DBaddr         string `yaml:"dbaddr"`
		DBname         string `yaml:"dbname"`
		Tablename      string `yaml:"tablename"`
	}
	Output struct {
		Prometheus      bool   `yaml:"prometheus"`
		PushGateway     bool   `yaml:"pushGateway"`
		PushGatewayAddr string `yaml:"pushGatewayAddr"`
		Period          int    `yaml:"period"`
	}
}

var globeCfg *RConfig

//mongodb 数据mode
type getpath struct {
	CallGetpath string `bson:"callGetpath"`
}

//gzip解压
func DoGzipUnCompress(compressSrc []byte) []byte {
	b := bytes.NewReader(compressSrc)
	var out bytes.Buffer
	r, _ := gzip.NewReader(b)
	io.Copy(&out, r)
	return out.Bytes()
}

//relay 节点数组
var relayid = [17]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 13, 14, 19, 20, 21, 23}

//prometheus var
var (
	node1h = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "relaydelay",
		Subsystem: "Histogram",
		Name:      "1",
		Help:      "node1",
		Buckets:   prometheus.LinearBuckets(10, 30, 3),
	},
		[]string{
			"IP",
			"RelayId",
		})
	node1s = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "relaydelay",
		Subsystem:  "Summary",
		Name:       "1",
		Help:       "node1",
		Objectives: map[float64]float64{0: 0.05, 0.5: 0.03, 1: 0.01},
	},
		[]string{
			"IP",
			"RelayId",
		})
	node2h = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "relaydelay",
		Subsystem: "Histogram",
		Name:      "2",
		Help:      "node2",
		Buckets:   prometheus.LinearBuckets(10, 30, 3),
	},
		[]string{
			"IP",
			"RelayId",
		})
	node2s = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "relaydelay",
		Subsystem:  "Summary",
		Name:       "2",
		Help:       "node2",
		Objectives: map[float64]float64{0: 0.05, 0.5: 0.03, 1: 0.01},
	},
		[]string{
			"IP",
			"RelayId",
		})
	node3h = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "relaydelay",
		Subsystem: "Histogram",
		Name:      "3",
		Help:      "node3",
		Buckets:   prometheus.LinearBuckets(10, 30, 3),
	},
		[]string{
			"IP",
			"RelayId",
		})
	node3s = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "relaydelay",
		Subsystem:  "Summary",
		Name:       "3",
		Help:       "node3",
		Objectives: map[float64]float64{0: 0.05, 0.5: 0.03, 1: 0.01},
	},
		[]string{
			"IP",
			"RelayId",
		})
	node4h = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "relaydelay",
		Subsystem: "Histogram",
		Name:      "4",
		Help:      "node4",
		Buckets:   prometheus.LinearBuckets(10, 30, 3),
	},
		[]string{
			"IP",
			"RelayId",
		})
	node4s = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "relaydelay",
		Subsystem:  "Summary",
		Name:       "4",
		Help:       "node4",
		Objectives: map[float64]float64{0: 0.05, 0.5: 0.03, 1: 0.01},
	},
		[]string{
			"IP",
			"RelayId",
		})
	node5h = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "relaydelay",
		Subsystem: "Histogram",
		Name:      "5",
		Help:      "node5",
		Buckets:   prometheus.LinearBuckets(10, 30, 3),
	},
		[]string{
			"IP",
			"RelayId",
		})
	node5s = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "relaydelay",
		Subsystem:  "Summary",
		Name:       "5",
		Help:       "node5",
		Objectives: map[float64]float64{0: 0.05, 0.5: 0.03, 1: 0.01},
	},
		[]string{
			"IP",
			"RelayId",
		})
	node6h = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "relaydelay",
		Subsystem: "Histogram",
		Name:      "6",
		Help:      "node6",
		Buckets:   prometheus.LinearBuckets(10, 30, 3),
	},
		[]string{
			"IP",
			"RelayId",
		})
	node6s = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "relaydelay",
		Subsystem:  "Summary",
		Name:       "6",
		Help:       "node6",
		Objectives: map[float64]float64{0: 0.05, 0.5: 0.03, 1: 0.01},
	},
		[]string{
			"IP",
			"RelayId",
		})
	node7h = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "relaydelay",
		Subsystem: "Histogram",
		Name:      "7",
		Help:      "node7",
		Buckets:   prometheus.LinearBuckets(10, 30, 3),
	},
		[]string{
			"IP",
			"RelayId",
		})
	node7s = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "relaydelay",
		Subsystem:  "Summary",
		Name:       "7",
		Help:       "node7",
		Objectives: map[float64]float64{0: 0.05, 0.5: 0.03, 1: 0.01},
	},
		[]string{
			"IP",
			"RelayId",
		})
	node8h = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "relaydelay",
		Subsystem: "Histogram",
		Name:      "8",
		Help:      "node8",
		Buckets:   prometheus.LinearBuckets(10, 30, 3),
	},
		[]string{
			"IP",
			"RelayId",
		})
	node8s = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "relaydelay",
		Subsystem:  "Summary",
		Name:       "8",
		Help:       "node8",
		Objectives: map[float64]float64{0: 0.05, 0.5: 0.03, 1: 0.01},
	},
		[]string{
			"IP",
			"RelayId",
		})
	node9h = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "relaydelay",
		Subsystem: "Histogram",
		Name:      "9",
		Help:      "node9",
		Buckets:   prometheus.LinearBuckets(10, 30, 3),
	},
		[]string{
			"IP",
			"RelayId",
		})
	node9s = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "relaydelay",
		Subsystem:  "Summary",
		Name:       "9",
		Help:       "node9",
		Objectives: map[float64]float64{0: 0.05, 0.5: 0.03, 1: 0.01},
	},
		[]string{
			"IP",
			"RelayId",
		})
	node10h = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "relaydelay",
		Subsystem: "Histogram",
		Name:      "10",
		Help:      "node10",
		Buckets:   prometheus.LinearBuckets(10, 30, 3),
	},
		[]string{
			"IP",
			"RelayId",
		})
	node10s = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "relaydelay",
		Subsystem:  "Summary",
		Name:       "10",
		Help:       "node10",
		Objectives: map[float64]float64{0: 0.05, 0.5: 0.03, 1: 0.01},
	},
		[]string{
			"IP",
			"RelayId",
		})
	node11h = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "relaydelay",
		Subsystem: "Histogram",
		Name:      "11",
		Help:      "node11",
		Buckets:   prometheus.LinearBuckets(10, 30, 3),
	},
		[]string{
			"IP",
			"RelayId",
		})
	node11s = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "relaydelay",
		Subsystem:  "Summary",
		Name:       "11",
		Help:       "node11",
		Objectives: map[float64]float64{0: 0.05, 0.5: 0.03, 1: 0.01},
	},
		[]string{
			"IP",
			"RelayId",
		})
	node13h = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "relaydelay",
		Subsystem: "Histogram",
		Name:      "13",
		Help:      "node13",
		Buckets:   prometheus.LinearBuckets(10, 30, 3),
	},
		[]string{
			"IP",
			"RelayId",
		})
	node13s = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "relaydelay",
		Subsystem:  "Summary",
		Name:       "13",
		Help:       "node13",
		Objectives: map[float64]float64{0: 0.05, 0.5: 0.03, 1: 0.01},
	},
		[]string{
			"IP",
			"RelayId",
		})
	node14h = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "relaydelay",
		Subsystem: "Histogram",
		Name:      "id14",
		Help:      "node14",
		Buckets:   prometheus.LinearBuckets(10, 30, 3),
	},
		[]string{
			"IP",
			"RelayId",
		})
	node14s = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "relaydelay",
		Subsystem:  "Summary",
		Name:       "id14",
		Help:       "node14",
		Objectives: map[float64]float64{0: 0.05, 0.5: 0.03, 1: 0.01},
	},
		[]string{
			"IP",
			"RelayId",
		})
	node19h = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "relaydelay",
		Subsystem: "Histogram",
		Name:      "19",
		Help:      "node19",
		Buckets:   prometheus.LinearBuckets(10, 30, 3),
	},
		[]string{
			"IP",
			"RelayId",
		})
	node19s = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "relaydelay",
		Subsystem:  "Summary",
		Name:       "19",
		Help:       "node19",
		Objectives: map[float64]float64{0: 0.05, 0.5: 0.03, 1: 0.01},
	},
		[]string{
			"IP",
			"RelayId",
		})
	node20h = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "relaydelay",
		Subsystem: "Histogram",
		Name:      "20",
		Help:      "node20",
		Buckets:   prometheus.LinearBuckets(10, 30, 3),
	},
		[]string{
			"IP",
			"RelayId",
		})
	node20s = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "relaydelay",
		Subsystem:  "Summary",
		Name:       "20",
		Help:       "node20",
		Objectives: map[float64]float64{0: 0.05, 0.5: 0.03, 1: 0.01},
	},
		[]string{
			"IP",
			"RelayId",
		})
	node21h = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "relaydelay",
		Subsystem: "Histogram",
		Name:      "21",
		Help:      "node21",
		Buckets:   prometheus.LinearBuckets(10, 30, 3),
	},
		[]string{
			"IP",
			"RelayId",
		})
	node21s = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "relaydelay",
		Subsystem:  "Summary",
		Name:       "21",
		Help:       "node21",
		Objectives: map[float64]float64{0: 0.05, 0.5: 0.03, 1: 0.01},
	},
		[]string{
			"IP",
			"RelayId",
		})
	node23h = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "relaydelay",
		Subsystem: "Histogram",
		Name:      "23",
		Help:      "node23",
		Buckets:   prometheus.LinearBuckets(10, 30, 3),
	},
		[]string{
			"IP",
			"RelayId",
		})
	node23s = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "relaydelay",
		Subsystem:  "Summary",
		Name:       "23",
		Help:       "node23",
		Objectives: map[float64]float64{0: 0.05, 0.5: 0.03, 1: 0.01},
	},
		[]string{
			"IP",
			"RelayId",
		})
)

//prometheus 注册
func Must() {
	prometheus.MustRegister(node1h)
	prometheus.MustRegister(node1s)
	prometheus.MustRegister(node2h)
	prometheus.MustRegister(node2s)
	prometheus.MustRegister(node3h)
	prometheus.MustRegister(node3s)
	prometheus.MustRegister(node4h)
	prometheus.MustRegister(node4s)
	prometheus.MustRegister(node5h)
	prometheus.MustRegister(node5s)
	prometheus.MustRegister(node6h)
	prometheus.MustRegister(node6s)
	prometheus.MustRegister(node7h)
	prometheus.MustRegister(node7s)
	prometheus.MustRegister(node8h)
	prometheus.MustRegister(node8s)
	prometheus.MustRegister(node9h)
	prometheus.MustRegister(node9s)
	prometheus.MustRegister(node10h)
	prometheus.MustRegister(node10s)
	prometheus.MustRegister(node11h)
	prometheus.MustRegister(node11s)
	prometheus.MustRegister(node13h)
	prometheus.MustRegister(node13s)
	prometheus.MustRegister(node14h)
	prometheus.MustRegister(node14s)
	prometheus.MustRegister(node19h)
	prometheus.MustRegister(node19s)
	prometheus.MustRegister(node20h)
	prometheus.MustRegister(node20s)
	prometheus.MustRegister(node21h)
	prometheus.MustRegister(node21s)
	prometheus.MustRegister(node23h)
	prometheus.MustRegister(node23s)

}

//给prometheus推送数据
func extract(ur map[float64]float64, ru map[float64]float64) error {
	for _, id := range relayid {
		if ur[id] != 0 {
			Observe(id, ur[id])

		}
		if ru[id] != 0 {
			Observe(id, ru[id])

		}
	}

	return nil
}

func Observe(id float64, delay float64) {
	switch id {
	case 1:
		node1s.WithLabelValues("210.51.168.108", "1").Observe(delay)
		node1h.WithLabelValues("210.51.168.108", "1").Observe(delay)
	case 2:
		node2s.WithLabelValues("114.112.74.12", "2").Observe(delay)
		node2h.WithLabelValues("114.112.74.12", "2").Observe(delay)
	case 3:
		node3s.WithLabelValues("175.102.21.33", "3").Observe(delay)
		node3h.WithLabelValues("175.102.21.33", "3").Observe(delay)
	case 4:
		node4s.WithLabelValues("175.102.8.227", "4").Observe(delay)
		node4h.WithLabelValues("175.102.8.227", "4").Observe(delay)
	case 5:
		node5s.WithLabelValues("122.13.78.226", "5").Observe(delay)
		node5h.WithLabelValues("122.13.78.226", "5").Observe(delay)
	case 6:
		node6s.WithLabelValues("125.88.254.159", "6").Observe(delay)
		node6h.WithLabelValues("125.88.254.159", "6").Observe(delay)
	case 7:
		node7s.WithLabelValues("125.211.202.28", "7").Observe(delay)
		node7h.WithLabelValues("125.211.202.28", "7").Observe(delay)
	case 8:
		node8s.WithLabelValues("222.171.242.142", "8").Observe(delay)
		node8h.WithLabelValues("222.171.242.142", "8").Observe(delay)
	case 9:
		node9s.WithLabelValues("123.138.91.24", "9").Observe(delay)
		node9h.WithLabelValues("123.138.91.24", "9").Observe(delay)
	case 10:
		node10s.WithLabelValues("124.116.176.115", "10").Observe(delay)
		node10h.WithLabelValues("124.116.176.115", "10").Observe(delay)
	case 11:
		node11s.WithLabelValues("221.7.112.74", "11").Observe(delay)
		node11h.WithLabelValues("221.7.112.74", "11").Observe(delay)
	case 13:
		node13s.WithLabelValues("220.249.119.217", "13").Observe(delay)
		node13h.WithLabelValues("220.249.119.217", "13").Observe(delay)
	case 14:
		node14s.WithLabelValues("61.183.245.140", "14").Observe(delay)
		node14h.WithLabelValues("61.183.245.140", "14").Observe(delay)
	case 19:
		node19s.WithLabelValues("103.25.23.121", "19").Observe(delay)
		node19h.WithLabelValues("103.25.23.121", "19").Observe(delay)
	case 20:
		node20s.WithLabelValues("103.25.23.122", "20").Observe(delay)
		node20h.WithLabelValues("103.25.23.122", "20").Observe(delay)
	case 21:
		node21s.WithLabelValues("223.111.205.86", "21").Observe(delay)
		node21h.WithLabelValues("223.111.205.86", "21").Observe(delay)
	case 23:
		node23s.WithLabelValues("223.111.205.90", "23").Observe(delay)
		node23h.WithLabelValues("223.111.205.90", "23").Observe(delay)
	default:
		fmt.Errorf("line:383 undefined relayId")
	}
}

//从mongedb获取以解码的getpath明文数据数组
func mongodbTogetpath(ip string, db string, table string) []string {

	looptime := int64(globeCfg.Output.Period) //minute
	session, err := mgo.Dial(ip)
	//session, err := mgo.Dial("127.0.0.1:27017")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	collection := session.DB(db).C(table)
	var nowtime int64
	nowtime = time.Now().UnixNano() / 1000000
	var min10time int64
	min10time = nowtime - looptime*1000
	var getpathresult []getpath

	//通过sid获取getpath日志
	err = collection.Find(bson.M{"insertTime": bson.M{"$gt": min10time, "$lt": nowtime}}).Select(bson.M{"callGetpath": 1}).All(&getpathresult)
	if err != nil {
		panic(err)
	}
	i := 0
	for _, v := range getpathresult {
		if v.CallGetpath != "" {
			i++
		}
	}
	var result2 []getpath = make([]getpath, i)
	i = 0
	for _, v := range getpathresult {
		if v.CallGetpath != "" {
			result2[i] = v
			i++
		}
	}
	i = 0
	// 存放未解码的getpsth
	j := len(result2)
	var getpathString []string = make([]string, j)
	for _, result := range result2 {
		getpathString[i] = getpathToString(result)
		i++
	}
	return getpathString
}

// 将getpath中加密的数据变成明码
func getpathToString(result2 getpath) string {
	var zsrcs []byte = nil
	//获取加密过得getpath后按\n分开后分别进行base64解码；后组合在进行gzip解码；后变为明码
	var b64str []string = strings.Split(result2.CallGetpath, "\n")
	for _, str := range b64str {
		strc, _ := base64.StdEncoding.DecodeString(str)
		zsrcs = append(zsrcs, strc...)
	}
	ugzip := DoGzipUnCompress(zsrcs)
	return string(ugzip)
}

// 解析解码后的getpath获取各relay节点延迟 并推送给prometheus
func getpathToMap(str string) {
	//存放delay
	urdelaymap := make(map[float64]float64)
	rudelaymap := make(map[float64]float64)
	var delay float64
	//解析ugzip json 获取relay延迟
	var jsonmap map[string]interface{}
	json.Unmarshal([]byte(str), &jsonmap)

	if v, ok := jsonmap["ur_link_info"]; ok {
		ur_link_info := v.(map[string]interface{})
		if v, ok := ur_link_info["U_R_self"]; ok {
			if v != nil {

				U_R_self := v.([]interface{})
				for _, v := range U_R_self {
					vmap := v.(map[string]interface{})
					if v, ok := vmap["delay"]; ok {
						delay = v.(float64)
					}
					if v, ok := vmap["relayID"]; ok {
						urdelaymap[v.(float64)] = delay
					}
				}
			}

		}
		if v, ok := ur_link_info["R_U_self"]; ok {
			if v != nil {
				R_U_self := v.([]interface{})

				for _, v := range R_U_self {
					vmap := v.(map[string]interface{})
					if v, ok := vmap["delay"]; ok {
						delay = v.(float64)
					}
					if v, ok := vmap["relayID"]; ok {
						rudelaymap[v.(float64)] = delay
					}
				}
			}

		}

	}

	extract(urdelaymap, rudelaymap)
}
func loadConfig() {
	cfgbuf, err := ioutil.ReadFile("cfg.yaml")
	if err != nil {
		panic("not found cfg.yaml")
	}
	rfig := RConfig{}
	err = yaml.Unmarshal(cfgbuf, &rfig)
	if err != nil {
		panic("invalid cfg.yaml")
	}
	globeCfg = &rfig
	fmt.Println("Load config -'cfg.yaml'- ok...")
}

func main() {

	loadConfig()                   //加载配置文件
	ip := globeCfg.Gw.DBaddr       //"103.25.23.89:60013"
	db := globeCfg.Gw.DBname       //"dataAnalysis_new"
	table := globeCfg.Gw.Tablename //"report_tab"
	//loop
	go func() {
		Must() //注册prometheus
		fmt.Println("Program startup ok...")
		//获取getpath明码数据
		getpathstring := mongodbTogetpath(ip, db, table)
		for _, v := range getpathstring {
			//获取延迟数据推送给prometheus
			getpathToMap(v)
		}
		//是否推送数据给PushGatway
		if globeCfg.Output.PushGateway {
			if err := push.FromGatherer("relaydelay", push.HostnameGroupingKey(), globeCfg.Output.PushGatewayAddr, prometheus.DefaultGatherer); err != nil {
				fmt.Println("FromGatherer:", err)
			}
		}
		time.Sleep(time.Duration(globeCfg.Output.Period) * time.Second)
	}()
	//设置prometheus监听的ip和端口
	if globeCfg.Output.Prometheus {
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			http.ListenAndServe(fmt.Sprintf("%s:%d", globeCfg.Gw.Addr, globeCfg.Gw.HttpListenPort), nil)
			fmt.Println("ip", globeCfg.Gw.Addr)
			fmt.Println("port", globeCfg.Gw.HttpListenPort)
		}()
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c)
	signal.Notify(c, os.Interrupt, os.Kill)
	s := <-c
	fmt.Println("exit", s)

}

package main

import (
	"bytes"
	"compress/gzip"
	"container/list"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"net/http"
	"os"
	"os/signal"
	"strconv"
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
	Relay struct {
		Relaynode          map[int64]string    `yaml:"relaynode"`
		HistogramOptsparam map[string]float64  `yaml:"histogramOptsparam"`
		SummaryOptsparam   map[float64]float64 `yaml:"summaryOptsparam"`
	}
}

var globeCfg *RConfig

//mongodb 数据mode
type getpath struct {
	CallGetpath string `bson:"callGetpath"`
}

type itime struct {
	InsertTime int64 `bson:"insertTime"`
}

//relay 节点数组 map
var relayidArray = list.New()
var relayidMap = make(map[int64]string)

//HistogramOpt参数
var HistogramOptsparamMap = make(map[string]float64)

//SummaryOpt参数
var SummaryOptsparamMap = make(map[float64]float64)

//上一次的数据库的最后插入时间
var lasttime int64 = 0

//gzip解压
func DoGzipUnCompress(compressSrc []byte) []byte {
	b := bytes.NewReader(compressSrc)
	var out bytes.Buffer
	r, _ := gzip.NewReader(b)
	io.Copy(&out, r)
	return out.Bytes()
}

//prometheus var
var (
	nodeh *(prometheus.HistogramVec)
	nodes *(prometheus.SummaryVec)
)

//prometheus 注册
func Must() {
	//	prometheus.MustRegister(nodeh)
	//	prometheus.MustRegister(nodes)

}

//给prometheus推送数据
func extract(ur map[int64]int64, ru map[int64]int64) error {

	for e := relayidArray.Front(); e != nil; e = e.Next() {

		id := e.Value.(int64)
		if ur[id] != 0 {
			Observe(id, ur[id])

		}
		if ru[id] != 0 {
			Observe(id, ru[id])

		}

	}

	return nil
}

func Observe(id int64, delay int64) {
	strid := strconv.FormatInt(id, 10)
	nodeh.WithLabelValues(strid, relayidMap[id]).Observe(float64(delay))
	nodes.WithLabelValues(strid, relayidMap[id]).Observe(float64(delay))

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

	var nowtime itime
	err = collection.Find(bson.M{}).Sort("-insertTime").Limit(1).Select(bson.M{"insertTime": 1}).One(&nowtime)

	var min10time int64
	if lasttime == 0 {
		min10time = nowtime.InsertTime - looptime*1000
	} else {
		min10time = lasttime
	}
	var getpathresult []getpath

	//通过sid获取getpath日志
	err = collection.Find(bson.M{"insertTime": bson.M{"$gt": min10time, "$lt": nowtime.InsertTime}, "callGetpath": bson.M{"$exists": true}}).Select(bson.M{"callGetpath": 1}).All(&getpathresult)
	if err != nil {
		panic(err)
	}
	// 存放未解码的getpsth
	i := 0
	j := len(getpathresult)
	var getpathString []string = make([]string, j)
	for _, result := range getpathresult {
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
	urdelaymap := make(map[int64]int64)
	rudelaymap := make(map[int64]int64)
	var delay int64
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
						delay = int64(v.(float64))

					}

					if v, ok := vmap["relayID"]; ok {

						urdelaymap[int64(v.(float64))] = delay
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

						delay = int64(v.(float64))
					}
					if v, ok := vmap["relayID"]; ok {

						rudelaymap[int64(v.(float64))] = delay
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
func init() {
	loadConfig() //加载配置文件
	relayidMap = globeCfg.Relay.Relaynode

	for k, _ := range relayidMap {

		relayidArray.PushBack(k)

	}

	HistogramOptsparamMap = globeCfg.Relay.HistogramOptsparam
	SummaryOptsparamMap = globeCfg.Relay.SummaryOptsparam

	nodeh = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "relaydelay",
		Subsystem: "Histogram",
		Name:      "relay",
		Help:      "nodeh",
		Buckets:   prometheus.LinearBuckets(HistogramOptsparamMap["start"], HistogramOptsparamMap["width"], int(HistogramOptsparamMap["count"])),
	},
		[]string{
			"IP",
			"RelayId",
		})

	nodes = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "relaydelay",
		Subsystem:  "Summary",
		Name:       "relay",
		Help:       "nodes",
		Objectives: SummaryOptsparamMap,
	},
		[]string{
			"IP",
			"RelayId",
		})

	prometheus.MustRegister(nodeh)
	prometheus.MustRegister(nodes)

}
func main() {

	ip := globeCfg.Gw.DBaddr       //"103.25.23.89:60013"
	db := globeCfg.Gw.DBname       //"dataAnalysis_new"
	table := globeCfg.Gw.Tablename //"report_tab"
	//loop
	go func() {
		Must() //注册prometheus
		fmt.Println("Program startup ok...")
		//获取getpath明码数据
		for {

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
		}
	}()
	//设置prometheus监听的ip和端口
	if globeCfg.Output.Prometheus {

		go func() {
			fmt.Println("ip", globeCfg.Gw.Addr)
			fmt.Println("port", globeCfg.Gw.HttpListenPort)
			http.Handle("/metrics", promhttp.Handler())
			http.ListenAndServe(fmt.Sprintf("%s:%d", globeCfg.Gw.Addr, globeCfg.Gw.HttpListenPort), nil)

		}()
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c)
	signal.Notify(c, os.Interrupt, os.Kill)
	s := <-c
	fmt.Println("exit", s)

}

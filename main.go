package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/tkanos/gonfig"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)
//структура конфига
type Configuration struct {
	ConnectionString string
	Timeout time.Duration
}
var configuration = Configuration{}
//структура получения курсов из источника
type Curresp struct {
	Base string `json:"base"`
	Rates map[string]float64 `json:"rates"`
}
//структура курса для 2 валют
type TargetToBase struct {
	Base string
	Target string
	Sum float64
	Result float64
}
//http клиент
var myClient = &http.Client{Timeout: configuration.Timeout * time.Second}
//функция получения курсов из источника
func getJson(target interface{}) error {
	r, err := myClient.Get(configuration.ConnectionString)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			panic(err.Error())
		}
	}()
	return json.Unmarshal(body,target)
}
//функция вывода ошибки
func sendError(w http.ResponseWriter,statusCode int, errorMsg string){
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(`{"error": "` + errorMsg + `"}`))
}
// функция рассчёта target по base
func setTarget(r *http.Request, sum float64,t *TargetToBase, c *Curresp) error {
	if c.Rates[strings.ToUpper(mux.Vars(r)["base"])]*c.Rates[strings.ToUpper(mux.Vars(r)["target"])] != 0 {
		t.Base = strings.ToUpper(mux.Vars(r)["base"])
		t.Target = strings.ToUpper(mux.Vars(r)["target"])
		t.Sum = sum
		t.Result = sum * c.Rates[t.Base] / c.Rates[t.Target]
	} else {
		return http.ErrBodyNotAllowed
	}
	return nil
}
//функия получения курсов всех валют к <base>
func getBase(w http.ResponseWriter, r *http.Request)  {
	w.Header().Set("Content-Type", "application/json")
	cur := new(Curresp)
	err := getJson(cur)
	if err != nil {
		sendError(w,400,"Can't get currency")
		return
	}
	baseName := strings.ToUpper(mux.Vars(r)["base"])
	baseVal := cur.Rates[baseName]
	if cur.Rates[baseName] == 0 {
		sendError(w,400,"Currency not found")
		return
	}
	cur.Base = baseName
	for k := range cur.Rates {
		cur.Rates[k] = baseVal / cur.Rates[k]
	}
	err = json.NewEncoder(w).Encode(cur)
	if err != nil {
		sendError(w,400,"JSON encode error")
		return
	}
}
// функция получения курса валюты <target> по отношению к <base>
func getTarget(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	cur := new(Curresp)
	err := getJson(cur)
	if err != nil {
		sendError(w,400,"Can't get currency'")
		return
	}
	target := new(TargetToBase)

	err = setTarget(r,1,target,cur)
	if err != nil {
		sendError(w,400,"Currency mismatch")
		return
	}
	err = json.NewEncoder(w).Encode(target)
	if err != nil {
		sendError(w,400,"JSON encode error")
		return
	}
}
// функция расчёта стоимости валюты в <target> по отношению к <base> в объёме <sum>
func getSum(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	cur := new(Curresp)
	err := getJson(cur)
	if err != nil {
		sendError(w,400,"Can't get currency")
		return
	}
	target := new(TargetToBase)
	sum, err := strconv.ParseFloat(mux.Vars(r)["sum"], 64)
	if err != nil {
		sendError(w,400,"Amount is not valid")
		return
	}
	err = setTarget(r,sum,target,cur)
	if err != nil {
		sendError(w,400,"Currency mismatch")
		return
	}
	err = json.NewEncoder(w).Encode(target)
	if err != nil {
		sendError(w,400,"JSON encode error")
		return
	}
}
func main() {
	err := gonfig.GetConf("tsconfig.json", &configuration)
	if err != nil {
		panic(err.Error())
	}
	log.Println("Server start")
	r := mux.NewRouter()
	r.HandleFunc("/currencies/{base}", getBase).Methods("GET")
	r.HandleFunc("/currencies/{base}/{target}", getTarget).Methods("GET")
	r.HandleFunc("/currencies/{base}/{target}/{sum}", getSum).Methods("GET")
	log.Fatal(http.ListenAndServe(":8000", r))
}
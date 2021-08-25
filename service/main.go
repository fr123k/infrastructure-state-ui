package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

func (w *TFVarValueWrapper) UnmarshalJSON(data []byte) error {
	w.Value = string(data)
	return nil
}

func (w TFVarValueWrapper) MarshalJSON() ([]byte, error) {
	return []byte(w.Value), nil
}

type TFVarValueWrapper struct {
	Value string `json:”-”`
}

type TFVar struct {
	Value TFVarValueWrapper `json:"value"`
}

type TFPlans struct {
	Plans []string `json:"plans"`
}

type Change struct {
	Actions      []string               `json:"actions"`
	Before       map[string]interface{} `json:"before"`
	After        map[string]interface{} `json:"after"`
	AfterUnknown map[string]interface{} `json:"after_unknown"`
}

type ResourceChange struct {
	Adress       string `json:"address"`
	Mode         string `json:"mode"`
	Type         string `json:"type"`
	Name         string `json:"name"`
	ProviderName string `json:"provider_name"`
	Change       Change `json:"change"`
}

type TFPlanMeta struct {
	Workspace string `json:"workspace"`
	Project   string `json:"project"`
	Date      string `json:"date"`
	CommitId  string `json:"commit_id"`
	Version   string `json:"version"`
	//TODO rename source to something more meaningful
	//Where that plan was produced.
	Source    string `json:"source"`
	SourceURL string `json:"source_url"`
}

type TFPlan struct {
	FormnatVersion string `json:"format_version"`
	TFVersion      string `json:"terraform_version"`
	//TODO the variable contains the client_id and client_secret of the azure remote backend storage
	//TODO it also contains the github token
	//Variables       map[string]TFVar `json:"variables"`
	ResourceChanges []ResourceChange `json:"resource_changes"`
	Meta            TFPlanMeta       `json:"meta"`
}

type TFSummary struct {
	TFPlan  TFPlan  `json:"plan"`
	Summary Summary `json:"summary"`
}

type SummaryChange struct {
	Action    string   `json:"action"`
	Count     int64    `json:"count"`
	Resources []string `json:"resources"`
}

type Summary struct {
	Changes   map[string]SummaryChange `json:"changes"`
	Resources int                      `json:"resources"`
	State     int                      `json:"state"`
}

type Configuration struct {
	ApplicationToken string
}

var cfg Configuration

func prettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func inc(i *int64) int64 { *i++; return *i }

//TODO refactor the getSummary/getPlan/getChanges
func getSummary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	project := params["project"]

	fmt.Printf("Received call %s\n", r.URL.Path)

	jsonFile := readJson(project)

	// read our opened xmlFile as a byte array.
	byteValue, err := ioutil.ReadAll(jsonFile)

	defer jsonFile.Close()

	if err != nil {
		fmt.Println(err)
	}

	// we initialize our Users array
	var tfPlan TFPlan

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'users' which we defined above
	json.Unmarshal(byteValue, &tfPlan)

	tfPlan.Meta = TFPlanMeta{
		Workspace: "default",
		Project:   "Project",
		Date:      "Today",
		Source:    "Google",
		SourceURL: "https://google.com",
	}

	var tfSummary TFSummary

	tfSummary.TFPlan = tfPlan
	tfSummary.Summary = Summary{Changes: make(map[string]SummaryChange),
		Resources: len(tfPlan.ResourceChanges)}

	for _, rChange := range tfPlan.ResourceChanges {
		for _, action := range rChange.Change.Actions {
			if sChange, ok := tfSummary.Summary.Changes[action]; ok {
				inc(&sChange.Count)
				sChange.Action = action
				sChange.Resources = append(sChange.Resources, rChange.Adress)
				tfSummary.Summary.Changes[action] = sChange
			} else {
				sChange := SummaryChange{Action: action,
					Count:     1,
					Resources: []string{rChange.Adress}}
				tfSummary.Summary.Changes[action] = sChange
			}
		}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(tfSummary)
	return
}

func getPlanFiles() []string {
	var files []string

	root := "/plans"
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		files = append(files, filepath.Base(path))
		return nil
	})
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		fmt.Println(file)
	}
	return files
}

func getPlans(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	fmt.Printf("Received call %s\n", r.URL.Path)

	jsonFiles := getPlanFiles()

	// we initialize our Users array
	plans := TFPlans{Plans: jsonFiles}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "    ")
	enc.Encode(plans)
	return
}

func getPlan(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	project := params["project"]

	fmt.Printf("Received call %s\n", r.URL.Path)

	jsonFile := readJson(project)

	// read our opened xmlFile as a byte array.
	byteValue, err := ioutil.ReadAll(jsonFile)

	defer jsonFile.Close()

	if err != nil {
		fmt.Println(err)
	}

	// we initialize our Users array
	var tfPlan TFPlan

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'users' which we defined above
	json.Unmarshal(byteValue, &tfPlan)

	tfPlan.Meta = TFPlanMeta{
		Workspace: "default",
		Project:   "Project",
		Date:      "Today",
		Source:    "Google",
		SourceURL: "https://google.com",
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "    ")
	enc.Encode(tfPlan)
	return
}

func reset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	fmt.Printf("Received call %s\n", r.URL.Path)

	dir, _ := ioutil.ReadDir("/plans")
	for _, d := range dir {
		os.RemoveAll(path.Join([]string{"plans", d.Name()}...))
	}

	return
}

func createPlan(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	project := params["project"]
	workspace := params["workspace"]
	version := params["version"]

	var plan TFPlan

	fmt.Printf("Received call %s\n", r.URL.Path)

	err := json.NewDecoder(r.Body).Decode(&plan)

	if nil != err {
		// Simplified
		log.Println(err)
		return
	}

	file, err := json.MarshalIndent(plan, "", " ")

	if nil != err {
		// Simplified
		log.Println(err)
		return
	}

	fileName := "/plans/" + project + "_" + workspace + "_" + version

	dirName := filepath.Dir(fileName)
	err = os.MkdirAll(dirName, 0644)
	if err != nil {
		// Simplified
		log.Println(err)
		return
	}

	err = ioutil.WriteFile(fileName, file, 0644)
	if err != nil {
		// Simplified
		log.Println(err)
		return
	}
	//TODO structure for retrieving plan for storage
	//TODO add storage simple json file storeage for example https://github.com/nanobox-io/golang-scribble

	return
}

func bin(i int) string {
	i64 := int64(i)
	return strconv.FormatInt(i64, 2) // base 2 for binary
}

func bin2int(binArStr []string) int {
	var binStr string
	for _, str := range binArStr {
		binStr = binStr + str
	}

	// base 2 for binary
	result, _ := strconv.ParseInt(binStr, 2, 64)
	return int(result)
}

func getChanges(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	project := params["project"]

	fmt.Printf("Received call %s\n", r.URL.Path)

	jsonFile := readJson(project)

	// read our opened xmlFile as a byte array.
	byteValue, err := ioutil.ReadAll(jsonFile)

	defer jsonFile.Close()

	if err != nil {
		fmt.Println(err)
	}

	// we initialize our Users array
	var tfPlan TFPlan

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'users' which we defined above
	json.Unmarshal(byteValue, &tfPlan)

	var tfPlanResult TFPlan

	if (tfPlan.Meta == TFPlanMeta{}) {
		tfPlan.Meta = TFPlanMeta{Workspace: "default",
			Project:   "empty",
			Date:      "empty",
			Source:    "empty",
			SourceURL: "empty"}
	}

	tfPlanResult.Meta = tfPlan.Meta
	tfPlanResult.FormnatVersion = tfPlan.FormnatVersion
	tfPlanResult.TFVersion = tfPlan.TFVersion
	//tfPlanResult.Variables = tfPlan.Variables
	tfPlanResult.ResourceChanges = []ResourceChange{}
	state := []string{"0", "0", "0", "0"}
	for _, rChange := range tfPlan.ResourceChanges {
		if contains(rChange.Change.Actions, "no-op") != true {
			tfPlanResult.ResourceChanges = append(tfPlanResult.ResourceChanges, rChange)
		}

		if contains(rChange.Change.Actions, "no-op") == true {
			state[3] = "1"
		}
		if contains(rChange.Change.Actions, "create") == true {
			state[2] = "1"
		}
		if contains(rChange.Change.Actions, "update") == true {
			state[1] = "1"
		}
		if contains(rChange.Change.Actions, "delete") == true {
			state[0] = "1"
		}
	}
	var tfSummary TFSummary

	tfSummary.TFPlan = tfPlanResult
	tfSummary.Summary = Summary{State: bin2int(state),
		Changes:   make(map[string]SummaryChange),
		Resources: len(tfPlan.ResourceChanges)}

	for _, rChange := range tfPlanResult.ResourceChanges {
		for _, action := range rChange.Change.Actions {
			if sChange, ok := tfSummary.Summary.Changes[action]; ok {
				inc(&sChange.Count)
				sChange.Action = action
				sChange.Resources = append(sChange.Resources, rChange.Adress)
				tfSummary.Summary.Changes[action] = sChange
			} else {
				sChange := SummaryChange{Action: action,
					Count:     1,
					Resources: []string{rChange.Adress}}
				tfSummary.Summary.Changes[action] = sChange
			}
		}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(tfSummary)
	return
}

//Initialize the Configuration struct by reading the values for it from the environment variables.
func Init() Configuration {
	authToken, ok := os.LookupEnv("AUTH_TOKEN")
	if ok {
		return Configuration{
			ApplicationToken: authToken,
		}
	}
	return Configuration{}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return defaultValue
	}
	return value
}

// Middleware function, which will be called for each request
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")

		splitToken := strings.Split(auth, " ")
		if len(splitToken) != 2 {
			log.Printf("Authorization header malformed '%s'\n", auth)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		authType := strings.TrimSpace(splitToken[0])
		if authType != "Bearer" {
			log.Printf("Only Authorization header type 'Bearer' is supported not '%s'\n", authType)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		b64Token := strings.TrimSpace(splitToken[1])
		token, err := base64.StdEncoding.DecodeString(b64Token)
		if err != nil {
			log.Printf("Authorization header token base64 malformed '%s'\n", b64Token)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		if string(token) == cfg.ApplicationToken {
			next.ServeHTTP(w, r)
		} else {
			log.Printf("Authentication failed secret mismatch \n")
			http.Error(w, "Forbidden", http.StatusForbidden)
		}
		return
	})
}

func readJson(fileName string) *os.File {
	// Open our jsonFile
	jsonFile, err := os.Open("/plans/" + fileName)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Successfully Opened " + fileName)
	// defer the closing of our jsonFile so that we can parse it later on
	return jsonFile
}

func IndexHandler(entrypoint string) func(w http.ResponseWriter, r *http.Request) {
	fn := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, entrypoint)
	}
	return http.HandlerFunc(fn)
}

func main() {
	router := mux.NewRouter()

	//TODO define timeout for consider when counter less then total the missing vms as dead and trigger an alert
	cfg = Init()

	//TODO add workspace handling
	api := router.PathPrefix("/api/").Subrouter()
	api.HandleFunc("/plan/{project}/summary", getSummary).Methods("GET")
	api.HandleFunc("/plan/{project}/changes", getChanges).Methods("GET")
	api.HandleFunc("/plan/{project}", getPlan).Methods("GET")
	api.HandleFunc("/plan", getPlans).Methods("GET")

	api.HandleFunc("/plan/{project}/workspace/{workspace}/version/{version}", createPlan).Methods("POST")

	api.HandleFunc("/admin/reset", reset).Methods("DELETE")

	router.PathPrefix("/static").Handler(http.FileServer(http.Dir("dist/")))
	// Catch-all: Serve our JavaScript application's entry-point (index.html).
	router.PathPrefix("/").HandlerFunc(IndexHandler("dist/index.html"))

	if len(cfg.ApplicationToken) > 0 {
		router.Use(Middleware)
	}

	log.Fatal(http.ListenAndServe(":"+getEnv("PORT", "8080"), router))
}

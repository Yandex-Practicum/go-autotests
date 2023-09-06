package pages

import (
	"fmt"
	shserv "github.com/Yandex-Practicum/go-autotests/internal/services/shurlService"
	"io/ioutil"
	"net/http"
	"strings"
)

var links = make(map[string]string)

func HandleShurlPage(res http.ResponseWriter, req *http.Request) {
	//res.Header().Set("Access-Control-Allow-Origin", "*")
	if req.Method == http.MethodPost { // Добавить ещё условие проверки длинности

		res.Header().Set("content-type", "text/plain") // wow, http.ContentTypeText doesn't work

		res.WriteHeader(http.StatusCreated) // 201
		hash := shserv.EvaluateHashAndReturn()

		b, err := ioutil.ReadAll(req.Body)
		if err != nil {
			panic(err)
		}
		links[hash] = string(b)
		_, err = res.Write([]byte(fmt.Sprintf(`%s:%s%s%s%s%s`, `http`, `/`, `/`, `localhost:8080`, `/`, hash)))
		if err != nil {
			return
		}
	} else if req.Method == http.MethodGet { // Добавить ещё условие проверки длинности
		//fmt.Println("gsdf")
		res.Header().Set("content-type", "text/plain") // wow, http.ContentTypeText doesn't work
		hash := strings.Split(req.URL.Path, "/")[1]
		res.Header().Set("Location", (fmt.Sprintf(links[hash])))
		res.WriteHeader(http.StatusTemporaryRedirect) // 307
		//_, err := res.Write([]byte(fmt.Sprintf(`http://%s`, links[hash])))
		//if err != nil {
		//	return
		//}
	} else {
		res.WriteHeader(http.StatusBadRequest) // 201
	}
}

//func GetLinkByHashAndRedirect(res http.ResponseWriter, req *http.Request) {
//	res.Header().Set("Access-Control-Allow-Origin", "*")
//	res.Header().Set("content-type", "text/plain")
//	res.WriteHeader(http.StatusTemporaryRedirect) // 307
//
//}

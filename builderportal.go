package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	//	"c3m/apps/tmplbuilder/portal/controllers/template"

	"flag"

	"github.com/spf13/viper"
	"github.com/tidusant/c3m-common/c3mcommon"
	"github.com/tidusant/c3m-common/log"
	"github.com/tidusant/c3m-common/mycrypto"
	"github.com/tidusant/chadmin-repo/models"

	//"io"
	rpb "github.com/tidusant/chadmin-repo/builder"
	rpsex "github.com/tidusant/chadmin-repo/session"

	"net/http"
	"net/url"

	//	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

var (
	user models.User
)

func init() {

}

func main() {
	var port int
	var debug bool
	//fmt.Println(mycrypto.Encode("abc,efc", 5))
	flag.IntVar(&port, "port", 7084, "help message for flagname")
	flag.BoolVar(&debug, "debug", false, "Indicates if debug messages should be printed in log files")
	flag.Parse()

	logLevel := log.DebugLevel
	if !debug {
		logLevel = log.InfoLevel
		gin.SetMode(gin.ReleaseMode)
	}

	log.SetOutputFile(fmt.Sprintf("templateportal-"+strconv.Itoa(port)), logLevel)
	defer log.CloseOutputFile()
	log.RedirectStdOut()

	log.Infof("running with port:" + strconv.Itoa(port))

	//check auth

	//init config

	router := gin.Default()

	router.POST("/:name", func(c *gin.Context) {

		strrt := c3mcommon.Fake64()
		requestDomain := c.Request.Header.Get("Origin")
		allowDomain := c3mcommon.CheckDomain(requestDomain)
		c.Header("Access-Control-Allow-Origin", "*")
		if allowDomain != "" {
			c.Header("Access-Control-Allow-Origin", requestDomain)
			c.Header("Access-Control-Allow-Headers", "access-control-allow-origin, access-control-allow-headers,access-control-allow-credentials")
			c.Header("Access-Control-Allow-Credentials", "true")

			if rpsex.CheckRequest(c.Request.URL.Path, c.Request.UserAgent(), c.Request.Referer(), c.Request.RemoteAddr, "POST") {
				rs := myRoute(c, allowDomain)
				b, _ := json.Marshal(rs)
				strrt = string(b)
				strrt = mycrypto.EncodeLight1(strrt, 5)
			} else {
				log.Debugf("check request error")

			}

		} else {
			log.Debugf("Not allow " + requestDomain)

		}
		c.String(http.StatusOK, strrt)
	})
	router.Run(":" + strconv.Itoa(port))

}

func myRoute(c *gin.Context, requestDomain string) models.RequestResult {

	name := c.Param("name")
	key := ""
	data := ""
	log.Debugf("key: %s", name)
	name = mycrypto.DecodeBK(name, "name")
	urls := strings.Split(name, "|")
	name = urls[0]
	session := ""
	if len(urls) > 1 {
		session = urls[1]
	}

	if name == "submit" {
		key = c.PostForm("key")
		if key != "" {
			key = mycrypto.Decode(key)
		}
		data = c.PostForm("data")
	} else {

		body, _ := ioutil.ReadAll(c.Request.Body)

		m, _ := url.ParseQuery(string(body))

		if len(m["key"]) > 0 {
			key = m["key"][0]
		}

		if len(m["data"]) > 0 {
			data = m["data"][0]
		}

	}

	//userIP, _, _ := net.SplitHostPort(c.Request.RemoteAddr)

	// RPCname := args[0]
	// if rpcname != "" {
	// 	RPCname = rpcname
	// }
	//return c3mcommon.ReturnJsonMessage("1", "", "", sample.Loaddata())
	userid := ""
	shopid := ""
	if requestDomain == viper.GetString("config.clienttemplatedomain") {
		if key == "" {
			// try to get key from data
			if data != "" {
				tmp := mycrypto.Decode(data)
				qrs, _ := url.ParseQuery(tmp)
				key = qrs.Get("key")
				log.Debugf("%s", key)
			}
		}
		userid = checkauth(key)
	} else {
		//check auth
		if session != "" {
			request := "aut|" + session
			rs := c3mcommon.RequestMainService(request, "POST", "aut")
			if rs.Status != "1" {
				return rs
			}
			logininfo := ""
			json.Unmarshal([]byte(rs.Data), &logininfo)
			shopargs := strings.Split(logininfo, "[+]")

			userid = shopargs[0]
			if len(shopargs) > 1 {
				shopid = shopargs[1]
			}

		}
	}

	//RPC call

	if userid == "" {
		return c3mcommon.ReturnJsonMessage("-1", "authorize fail", "", "")
	}

	if name == "loaddata" {
		//st := Loaddata()
		//log.Debugf("sample.Loaddata %s", st)
		//log.Debugf("done")
		return c3mcommon.ReturnJsonMessage("1", "", "", Loaddata())
	} else if name == "submit" {
		filetest, _ := c.FormFile("file")

		filetmp, _ := filetest.Open()

		f, err := os.OpenFile("templates/"+filetest.Filename, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			panic(err) //please dont
		}
		defer f.Close()
		io.Copy(f, filetmp)
		return SaveTemplates(userid, mycrypto.DecodeBK(data, name))
	} else if name == "loadtemplate" {
		return LoadTemplates(userid)
	} else if name == "getalltemplate" {
		return GetAllTemplate()
	} else if name == "installtemplate" {
		return InstallTemplate(shopid, mycrypto.Decode(data))
	} else if name == "activetemplate" {
		return ActiveTemplate(shopid, mycrypto.Decode(data))
	} else if name == "gettemplateconfig" {
		return GetTemplateConfig(shopid, mycrypto.Decode(data))
	} else if name == "savetemplateconfig" {
		return SaveTemplateConfig(shopid, mycrypto.Decode(data))
	} else if name == "gettemplateresource" {
		return GetTemplateResource(shopid, mycrypto.Decode(data))
	} else if name == "savetemplateresource" {
		return SaveTemplateResource(shopid, mycrypto.Decode(data))
	} else if name == "createbuild" {

		return CreateBuild(shopid, mycrypto.Decode(data))
	}

	return models.RequestResult{}
}

func checkauth(key string) string {

	user = rpb.AuthByKey(key)

	return user.ID.Hex()
}

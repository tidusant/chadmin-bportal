package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"github.com/tidusant/c3m-common/c3mcommon"
	"github.com/tidusant/c3m-common/log"
	"io/ioutil"
	"path/filepath"
	"time"
)

func Loaddata() string {
	//load data
	//demoshop := rpch.GetDemoShop()
	temprootpath := viper.GetString("config.templatepath")
	timestart := time.Now()
	strrt := "{\"ShopTitle\":\"Demo title\",\"ShopDescription\":\"Demo description\"" // ,\"Prods\":"

	strrt += ",\"Prods\":"

	//get demodata
	request := "DemoData"
	rs := c3mcommon.RequestMainService(request, "POST", viper.GetString("config.demouser")+","+viper.GetString("config.demouserpass"))
	if rs.Status != "1" {
		strrt += "[]"
	} else {
		logininfo := ""
		json.Unmarshal([]byte(rs.Data), &logininfo)
		log.Debugf("data: %v", logininfo)
		//info, _ := json.Marshal(prods)
		strrt += string(logininfo)
	}

	// strrt += ",\"News\":"

	// items := rpch.GetDemoNews()
	// info, _ = json.Marshal(items)
	// strrt += string(info)

	// strrt += ",\"NewsCats\":"

	// newscats := rpch.GetDemoNewsCats()
	// info, _ = json.Marshal(newscats)
	// strrt += string(info)

	//get corejs

	var jsbuffer bytes.Buffer
	tempscript := temprootpath + "/scripts"
	//bottomscriptjs
	jslibfiles, _ := ioutil.ReadDir(tempscript + "/bottomscript")
	for _, f := range jslibfiles {
		if !f.IsDir() {
			if filepath.Ext(f.Name()) == ".js" {
				b, err := ioutil.ReadFile(tempscript + "/bottomscript/" + f.Name())
				if err != nil {
					c3mcommon.CheckError(fmt.Sprintf("cannot read file %s!", f.Name()), err)
					continue
				}
				str := string(b)
				jsbuffer.WriteString("\n" + str)

			}
		}
	}
	//core js file
	jslibfiles, _ = ioutil.ReadDir(tempscript + "/core")
	for _, f := range jslibfiles {
		if !f.IsDir() {
			if filepath.Ext(f.Name()) == ".js" && f.Name() != "index.js" && f.Name() != "server.js" {
				b, err := ioutil.ReadFile(tempscript + "/core/" + f.Name())
				if err != nil {
					c3mcommon.CheckError(fmt.Sprintf("cannot read file %s!", f.Name()), err)
					continue
				}
				str := string(b)
				jsbuffer.WriteString("\n" + str)

			}
		}
	}
	b, _ := ioutil.ReadFile(tempscript + "/core/index.js")
	jsbuffer.WriteString("\n{{models}}\n" + string(b))

	strrt += ",\"corejs\":"
	corejs := make(map[string]string)
	corejs["data"] = jsbuffer.String()

	info, _ := json.Marshal(corejs)
	strrt += string(info)

	log.Debugf("loaddata time:%s", time.Since(timestart))
	strrt += "}"
	return strrt
}

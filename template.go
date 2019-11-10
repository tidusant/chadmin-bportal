package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/spf13/viper"
	rpb "github.com/tidusant/chadmin-repo/builder"

	"github.com/tidusant/chadmin-repo/models"

	"github.com/tidusant/c3m-common/c3mcommon"

	"encoding/json"
	"os"
	"strconv"
	"time"

	"github.com/tidusant/c3m-common/log"
	"github.com/tidusant/c3m-common/mystring"

	"gopkg.in/mgo.v2/bson"
)

func SaveTemplates(userid, params string) models.RequestResult {
	timestart := time.Now()
	var templ models.TemplateSubmit
	var savetemplate models.Template
	err := json.Unmarshal([]byte(params), &templ)
	if !c3mcommon.CheckError("template parse json", err) {
		log.Debugf(params)
		return c3mcommon.ReturnJsonMessage("0", "create template fail", "", "")
	}
	//check template and status:
	//log.Debugf("params %s", params)
	//log.Debugf("templ %v", templ)

	//check resubmit
	var oldtemplate models.Template
	if templ.Code != "" {
		oldtemplate = rpb.GetTemplateByCode(templ.Code)
		if oldtemplate.UserID != userid {
			return c3mcommon.ReturnJsonMessage("0", "resubmit template fail", "", "")
		}
		savetemplate = oldtemplate
	}

	if savetemplate.Code == "" {
		//check template duplicate for new template
		if !rpb.CheckTemplateDup(templ.Title) {
			log.Errorf("Save template fail, userid %s: for template code %s - template existed", userid, templ.Title)
			return c3mcommon.ReturnJsonMessage("0", "Save template fail - template existed!", "", "")
		}
		//get code
		codes := rpb.GetAllTemplatesCode()
		newcode := ""
		for {
			newcode = mystring.RandString(4)
			if _, ok := codes[newcode]; !ok {
				break
			}
		}
		savetemplate.Title = templ.Title
		savetemplate.Code = newcode
		savetemplate.Created = time.Now().UTC().Add(time.Hour + 7)
		savetemplate.ID = bson.NewObjectId()
		savetemplate.UserID = userid
	} else {
		savetemplate = oldtemplate
	}

	savetemplate.Status = 1
	savetemplate.Modified = time.Now().UTC().Add(time.Hour + 7)

	//check screenshot
	// imagebytes, err := base64.StdEncoding.DecodeString(templ.Screenshot)
	// if len(imagebytes) < 512 {
	// 	return c3mcommon.ReturnJsonMessage("0", "screenshot is not correct", "", "")
	// }
	// filetype := http.DetectContentType(imagebytes[:512])
	// if filetype != "image/jpeg" {
	// 	log.Errorf("invalid screenshot image type: %s", filetype)
	// 	return c3mcommon.ReturnJsonMessage("0", "invalid screenshot image type", "", "")
	// }

	//create new template folder
	templateFolder := "templates/" + savetemplate.Code
	os.RemoveAll(templateFolder)

	os.Mkdir(templateFolder, 0777)
	//unzip
	zipfile := "templates/" + savetemplate.Title + ".zip"
	_, err = c3mcommon.Unzip(zipfile, templateFolder)
	os.Remove(zipfile)

	if !c3mcommon.CheckError("Unzip template "+savetemplate.Title, err) {
		return c3mcommon.ReturnJsonMessage("0", "create template fail: "+err.Error(), "", "")
	}
	code := rpb.SaveTemplate(savetemplate)
	if code == "" {
		if templ.Code == "" {
			os.RemoveAll(templateFolder)
		}
		return c3mcommon.ReturnJsonMessage("0", "create template fail", "", "")
	}
	log.Debug("SaveTemplate time:%s", time.Since(timestart))

	return c3mcommon.ReturnJsonMessage("1", "", "create template success", "")
}

func LoadTemplates(userid string) models.RequestResult {
	tmpls := rpb.GetTemplatesByUserId(userid)
	strrt := "{"
	if len(tmpls) > 0 {
		for _, c := range tmpls {
			strrt += "\"" + c.Title + "\":{\"Title\":\"" + c.Title + "\",\"Code\":\"" + c.Code + "\",\"Status\":" + strconv.Itoa(c.Status) + "},"
		}
		strrt = strrt[:len(strrt)-1]
	}
	strrt += "}"
	return c3mcommon.ReturnJsonMessage("1", "", "load template success", strrt)
}
func InstallTemplate(shopid, data string) models.RequestResult {
	args := strings.Split(data, "|")
	if len(args) < 2 {
		return c3mcommon.ReturnJsonMessage("0", "invalid data", "", "")
	}
	code := args[0]
	defaultlang := args[1]
	tmpl := rpb.GetTemplateByCode(code)
	// var isInstalled = false
	// for _, installedid := range tmpl.InstalledIDs {
	// 	if installedid == shopid {
	// 		isInstalled = true
	// 	}
	// }
	// if isInstalled {
	// 	b, _ := json.Marshal(tmpl)
	// 	return c3mcommon.ReturnJsonMessage("1", "", "already installed", string(b))
	// }

	//new config
	templfolder := viper.GetString("config.templatepath") + tmpl.Code + "/"
	inputname := "resources/config.txt"
	//read file
	b, err := ioutil.ReadFile(templfolder + inputname)
	if err != nil {
		return c3mcommon.ReturnJsonMessage("0", "", fmt.Sprintf("cannot read file %s!", inputname), "")
	}
	resourcesconfig := string(b)
	//convert to \n line
	resourcesconfig = strings.Replace(resourcesconfig, "\r\n", "\n", -1)
	resourcesconfig = strings.Replace(resourcesconfig, "\r", "\n", -1)
	var installedConfigs []string
	if resourcesconfig != "" {
		lines := strings.Split(resourcesconfig, "\n")
		for _, line := range lines {
			if len(line) == 0 || line[:1] == "#" {
				continue
			}
			cfgArr := strings.Split(line, "::")
			if len(cfgArr) > 2 {
				var config models.TemplateConfig
				config.ShopID = shopid
				config.TemplateCode = tmpl.Code
				config.Key = cfgArr[0]
				config.Value = cfgArr[2]
				config.Type = cfgArr[1]
				rpb.InsertTemplateConfig(config)
				installedConfigs = append(installedConfigs, config.Key)

			}
		}
	}
	//remove unused config
	rpb.RemoveUnusedTemplateConfig(shopid, tmpl, installedConfigs)

	//new resource
	inputname = "resources/lang.txt"
	//read file
	b, err = ioutil.ReadFile(templfolder + "/" + inputname)
	if err != nil {
		return c3mcommon.ReturnJsonMessage("0", "", fmt.Sprintf("cannot read file %s!", inputname), "")
	}
	resourceslang := string(b)
	//convert to \n line
	resourceslang = strings.Replace(resourceslang, "\r\n", "\n", -1)
	resourceslang = strings.Replace(resourceslang, "\r", "\n", -1)
	var installedResources []string
	if resourceslang != "" {
		lines := strings.Split(resourceslang, "\n")

		for _, line := range lines {
			if len(line) == 0 || line[:1] == "#" {
				continue
			}
			srcArr := strings.Split(line, "::")
			if len(srcArr) > 2 {
				srcKey := srcArr[0]
				srcVal := srcArr[2]
				langs := make(map[string]string)
				langs[defaultlang] = srcVal
				var resource models.Resource
				resource.ShopID = shopid
				resource.TemplateCode = tmpl.Code
				resource.Key = srcKey
				resource.Type = srcArr[1]
				resource.Value = langs
				rpb.InsertResource(resource)
				installedResources = append(installedResources, resource.Key)
			}
		}
	}
	rpb.RemoveUnusedTemplateResource(shopid, tmpl, installedResources)
	//get pages
	pageresources := make(map[string]map[string]string)

	folders, err := ioutil.ReadDir(templfolder + "/resources/pages")
	if err == nil {
		for _, d := range folders {
			if !d.IsDir() {
				continue
			}
			files, err := ioutil.ReadDir(templfolder + "/resources/pages/" + d.Name())
			if err == nil {
				fileresources := make(map[string]string)
				for _, f := range files {
					if f.Name()[len(f.Name())-4:] != ".txt" {
						continue
					}
					b, err := ioutil.ReadFile(templfolder + "/resources/pages/" + d.Name() + "/" + f.Name())
					c3mcommon.CheckError(fmt.Sprintf("cannot read file %s!", f.Name()), err)
					filecontent := string(b)
					//convert to \n line
					filecontent = strings.Replace(filecontent, "\r\n", "\n", -1)
					filecontent = strings.Replace(filecontent, "\r", "\n", -1)
					fileresources[strings.Replace(f.Name(), ".txt", "", 1)] = filecontent
				}
				pageresources[d.Name()] = fileresources
			}
		}
	}
	b, _ = json.Marshal(pageresources)
	tmpl.Pages = string(b)

	tmpl.InstalledIDs = append(tmpl.InstalledIDs, shopid)
	rpb.UpdateInstallID(tmpl.Code, tmpl.InstalledIDs)
	b, _ = json.Marshal(tmpl)
	return c3mcommon.ReturnJsonMessage("1", "", "install template success", string(b))
}
func CreateBuild(shopid, data string) models.RequestResult {
	var bs models.BuildScript
	err := json.Unmarshal([]byte(data), &bs)
	if !c3mcommon.CheckError("json parse createbuild fail", err) {
		log.Debugf("json parse createbuild fail:%s", data)
		return c3mcommon.ReturnJsonMessage("0", err.Error(), "", "")
	}
	bs.ShopId = shopid
	rpb.CreateBuild(shopid, bs)
	return c3mcommon.ReturnJsonMessage("1", "", "create build success", "")
}
func ActiveTemplate(shopid, data string) models.RequestResult {

	codes := strings.Split(data, ",")
	activecode := codes[0]
	oldcode := codes[1]

	tmpl := rpb.GetTemplateByCode(activecode)
	var isActived = false
	for _, activedid := range tmpl.ActiveIDs {
		if activedid == shopid {
			isActived = true
		}
	}
	if isActived {
		b, _ := json.Marshal(tmpl)
		return c3mcommon.ReturnJsonMessage("1", "", "already Actived", string(b))
	}

	tmpl.ActiveIDs = append(tmpl.ActiveIDs, shopid)
	rpb.UpdateActiveID(tmpl.Code, tmpl.ActiveIDs)

	//remove old template active
	oldtmpl := rpb.GetTemplateByCode(oldcode)
	var appendactiveid []string
	for _, activedid := range oldtmpl.ActiveIDs {
		if activedid != shopid {
			appendactiveid = append(appendactiveid, activedid)
		}
	}
	rpb.UpdateActiveID(oldtmpl.Code, appendactiveid)

	b, _ := json.Marshal(tmpl)
	return c3mcommon.ReturnJsonMessage("1", "", "install template success", string(b))
}
func GetTemplateConfig(shopid, data string) models.RequestResult {

	code := data

	tmplconfigs := rpb.GetTemplateConfigs(shopid, code)
	str := `{"TemplateConfigs":[`
	for _, cfg := range tmplconfigs {
		str += `{"Key":"` + cfg.Key + `","Type":"` + cfg.Type + `","Value":"` + cfg.Value + `"},`
	}
	if len(tmplconfigs) > 1 {
		str = str[:len(str)-1]
	}
	str += `],"BuildConfigs":`
	buildconf := rpb.GetBuildConfig(shopid)
	b, _ := json.Marshal(buildconf)
	str += string(b) + `}`

	return c3mcommon.ReturnJsonMessage("1", "", "install template success", str)
}
func SaveTemplateConfig(shopid, data string) models.RequestResult {

	PData := struct {
		Code            string
		BuildConfig     models.BuildConfig
		TemplateConfigs []struct {
			Key   string
			Value string
		}
	}{}

	json.Unmarshal([]byte(data), &PData)

	tmplconfigs := PData.TemplateConfigs
	var conf models.TemplateConfig
	conf.ShopID = shopid
	conf.TemplateCode = PData.Code
	for _, cfg := range tmplconfigs {
		if cfg.Key == "" {
			continue
		}
		conf.Key = cfg.Key
		conf.Value = cfg.Value
		rpb.SaveTemplateConfig(conf)
	}
	PData.BuildConfig.ShopId = shopid
	rpb.SaveConfigByShopId(PData.BuildConfig)

	return c3mcommon.ReturnJsonMessage("1", "", "save config success", "")
}
func GetTemplateResource(shopid, data string) models.RequestResult {

	code := data

	tmplrs := rpb.GetAllResource(code, shopid)
	str := `[`
	for _, cfg := range tmplrs {
		b, _ := json.Marshal(cfg.Value)
		str += `{"Key":"` + cfg.Key + `","Type":"` + cfg.Type + `","Value":` + string(b) + `},`
	}
	if len(tmplrs) > 1 {
		str = str[:len(str)-1]
	}
	str += `]`

	return c3mcommon.ReturnJsonMessage("1", "", "Get Template Resource success", str)
}
func SaveTemplateResource(shopid, data string) models.RequestResult {

	PData := struct {
		Code      string
		Resources []struct {
			Key   string
			Type  string
			Value map[string]string
		}
	}{}

	err := json.Unmarshal([]byte(data), &PData)
	log.Debugf("json save template: %s", data)
	if err != nil {
		return c3mcommon.ReturnJsonMessage("0", "Save Resource error:"+err.Error(), "", "")
	}

	items := PData.Resources
	var rs models.Resource
	rs.ShopID = shopid
	rs.TemplateCode = PData.Code
	for _, item := range items {
		if item.Key == "" {
			continue
		}
		rs.Key = item.Key
		rs.Value = item.Value
		rpb.SaveResource(rs)
	}

	return c3mcommon.ReturnJsonMessage("1", "", "Save Resource success", "")
}

func GetAllTemplate() models.RequestResult {
	//check session

	//userid := shopargs[0]
	// shopid := shopargs[1]
	templates := rpb.GetAllTemplates()
	b, _ := json.Marshal(templates)
	return c3mcommon.ReturnJsonMessage("1", "", "install template success", string(b))
}

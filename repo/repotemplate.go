package repo

import (
	"c3m/apps/tempman/portal/models"

	"github.com/tidusant/c3m-common/c3mcommon"

	"gopkg.in/mgo.v2/bson"
)

func GetTemplatesByUserId(userid string) []models.Template {
	var rt []models.Template
	col := db.C("templates")
	var cond bson.M
	if userid != "0" {
		cond = bson.M{"userid": userid}
	}

	err := col.Find(cond).All(&rt)
	c3mcommon.CheckError("GetTemplatesByUserId", err)
	return rt
}

func GetAllTemplatesCode() map[string]string {
	rt := make(map[string]string)
	col := db.C("templates")
	var cond bson.M
	var rs []models.Template
	err := col.Find(cond).All(&rs)
	c3mcommon.CheckError("GetAllTemplatesCode", err)
	for _, v := range rs {
		rt[v.Code] = v.Code
	}
	return rt
}

func CheckTemplateDup(templ models.Template) bool {
	count := 0
	col := db.C("templates")
	var cond bson.M
	cond = bson.M{"title": templ.Title}
	count, err := col.Find(cond).Count()
	if c3mcommon.CheckError("CheckTemplateDup", err) && count == 0 {
		return true
	}
	return false
}

func GetTemplatesByCode(code string) models.Template {
	var rt models.Template
	col := db.C("templates")
	var cond bson.M
	cond = bson.M{"code": code}
	err := col.Find(cond).One(&rt)
	c3mcommon.CheckError("GetTemplatesByCode", err)
	return rt
}

func SaveTemplate(newtmpl models.Template) string {
	col := db.C("templates")
	_, err := col.UpsertId(newtmpl.ID, newtmpl)
	c3mcommon.CheckError("UpsertId template", err)
	return newtmpl.Code
}

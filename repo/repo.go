package repo

import (
	"c3m/apps/tempman/portal/models"

	"github.com/tidusant/c3m-common/c3mcommon"
	"github.com/tidusant/c3m-common/log"

	"gopkg.in/mgo.v2/bson"

	"os"

	"gopkg.in/mgo.v2"
)

var (
	db *mgo.Database
)

func init() {
	log.Infof("init repo")
	strErr := ""
	db, strErr = c3mcommon.ConnectDB("chtemplate")
	if strErr != "" {
		log.Infof(strErr)
		os.Exit(1)
	}
}

func AuthByKey(key string) models.User {

	col := db.C("users")

	// if prod.Code {

	// 	err := col.Insert(prod)
	// 	c3mcommon.CheckError("product Insert", err)
	// } else {
	var rs models.User
	err := col.Find(bson.M{"keypair": key}).One(&rs)
	c3mcommon.CheckError("getcatbycode", err)
	return rs
}

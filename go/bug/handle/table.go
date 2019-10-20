package handle

import (
	"itflow/bug/bugconfig"
	"itflow/bug/buglog"
	"itflow/bug/model"
	"encoding/json"
	"github.com/hyahm/golog"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type passBug struct {
	Id          int      `json:"id"`
	Date        int64    `json:"date"`
	Remark      string   `json:"remark"`
	SelectUsers []string `json:"selectusers"`
	Status      string   `json:"status"`
	Code        int      `json:"statuscode"`
	User        string   `json:"user"`
	Mu          *sync.Mutex
}

func PassBug(w http.ResponseWriter, r *http.Request) {
	headers(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method == http.MethodPost {
		conn, nickname, err := logtokenmysql(r)
		errorcode := &errorstruct{}
		if err != nil {
			golog.Error(err.Error())
			if err == NotFoundToken {
				w.Write(errorcode.ErrorNotFoundToken())
				return
			}
			w.Write(errorcode.ErrorConnentMysql())
			return
		}
		defer conn.Db.Close()
		ub := &passBug{}

		// 获取参数
		ss, err := ioutil.ReadAll(r.Body)
		if err != nil {
			golog.Error(err.Error())
			w.Write(errorcode.ErrorGetData())
			return
		}

		err = json.Unmarshal(ss, ub)
		if err != nil {
			golog.Error(err.Error())
			w.Write(errorcode.ErrorParams())
			return
		}

		// 判断这个bug是不是自己的任务，只有自己的任务才可以转交
		var splist string
		var hasperm bool
		err = conn.GetOne("select spusers from bugs where id=?", ub.Id).Scan(&splist)
		myuid := strconv.FormatInt(bugconfig.CacheNickNameUid[nickname], 10)
		for _, v := range strings.Split(splist, ",") {
			if myuid == v {
				hasperm = true
				break
			}
		}
		if !hasperm {
			w.Write(errorcode.ErrorNoPermission())
			return
		}
		// 判断状态是否存在
		var sid int64
		var ok bool
		if sid, ok = bugconfig.CacheStatusSid[ub.Status]; !ok {
			w.Write(errorcode.ErrorKeyNotFound())
			return
		}
		idstr := make([]string, 0)
		for _, v := range ub.SelectUsers {
			var uid int64
			if uid, ok = bugconfig.CacheRealNameUid[v]; !ok {
				w.Write(errorcode.ErrorKeyNotFound())
				return
			}
			idstr = append(idstr, strconv.FormatInt(uid, 10))
		}

		ul := strings.Join(idstr, ",")
		//添加进information表, 应该要弄成事务,插入转交信息
		remarksql := "insert into informations(uid,bid,info,time) values(?,?,?,?)"
		_, err = conn.Insert(remarksql, bugconfig.CacheNickNameUid[nickname], ub.Id, ub.Remark, time.Now().Unix())
		if err != nil {
			golog.Error(err.Error())
			w.Write(errorcode.ErrorConnentMysql())
			return
		}
		//更改bug

		_, err = conn.Update("update bugs set sid=?,spusers=? where id=?", sid, ul, ub.Id)
		if err != nil {
			golog.Error(err.Error())
			w.Write(errorcode.ErrorConnentMysql())
			return
		}
		il := buglog.AddLog{
			Conn:     conn,
			Ip:       strings.Split(r.RemoteAddr, ":")[0],
			Classify: "bug",
		}
		err = il.Add(ub.Id, nickname, ub.SelectUsers)
		if err != nil {
			golog.Error(err.Error())
			w.Write(errorcode.ErrorConnentMysql())
			return
		}
		send, _ := json.Marshal(ub)
		w.Write(send)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

//func GetThisTask(w http.ResponseWriter, r *http.Request) {
//	headers(w, r)
//	if r.Method == http.MethodOptions {
//		w.WriteHeader(http.StatusOK)
//		return
//	}
//
//	if r.Method == http.MethodPost {
//		logger, conn, _, err := logtokenmysql(r)
//		defer conn.Db.Close()
//		gid, err := ioutil.ReadAll(r.Body)
//
//		if err != nil {
//			logger.ErrorLog("file:table.go,line:134,%v", err)
//			w.Write([]byte("fail"))
//			return
//
//		}
//		id, err := strconv.Atoi(string(gid))
//		if err != nil {
//			logger.ErrorLog("file:table.go,line:151,%v", err)
//			w.Write([]byte("fail"))
//			return
//		}
//		getaritclesql := "select status,spusers from bugs where id=?"
//
//		data, _, err := conn.SelectSlice_Slice(getaritclesql, id)
//
//		if err != nil {
//			logger.ErrorLog("file:table.go,line:210,%v", err)
//			w.Write([]byte("fail"))
//			return
//		}
//		senddata := &passtask{}
//		statusname, err := asset.StatusidGetName(data[0][0], conn)
//		if err != nil {
//			logger.ErrorLog("file:table.go,line:210,%v", err)
//			w.Write([]byte("fail"))
//			return
//		}
//		senddata.Status = statusname
//		senddata.Id = id
//		senddata.SelectUsers = strings.Split(data[0][0], ",")
//		senddata.Remark = ""
//		send, err := json.Marshal(senddata)
//		if err != nil {
//			logger.ErrorLog("file:table.go,line:73,%v", err)
//			w.Write([]byte("fail"))
//			return
//		}
//		w.Write(send)
//		return
//	}
//	w.WriteHeader(http.StatusNotFound)
//}

func TaskList(w http.ResponseWriter, r *http.Request) {
	headers(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == http.MethodPost {
		conn, name, err := logtokenmysql(r)
		errorcode := &errorstruct{}
		if err != nil {
			golog.Error(err.Error())
			if err == NotFoundToken {
				w.Write(errorcode.ErrorNotFoundToken())
				return
			}
			w.Write(errorcode.ErrorConnentMysql())
			return
		}
		defer conn.Db.Close()
		al := &model.AllArticleList{}

		uid := bugconfig.CacheNickNameUid[name]

		getaritclesql := "select id,createtime,importent,status,bugtitle,uid,level,pid,spusers from bugs where id in (select bid from userandbug where uid=?)  order by id desc "

		rows, err := conn.GetRows(getaritclesql, uid)

		if err != nil {
			golog.Error(err.Error())
			w.Write(errorcode.ErrorConnentMysql())
			return
		}
		for rows.Next() {
			sendlist := &model.ArticleList{}
			var statusid int64
			var spusers string
			var uid int64
			var pid int64
			rows.Scan(&sendlist.ID, &sendlist.Date, &sendlist.Importance, &statusid, &sendlist.Title, &uid, &sendlist.Level, &pid, &spusers)
			sendlist.Handle = formatUserlistToShow(spusers)
			sendlist.Status = bugconfig.CacheSidStatus[statusid]

			sendlist.Author = bugconfig.CacheUidRealName[uid]
			sendlist.Projectname = bugconfig.CachePidName[pid]

			al.Al = append(al.Al, sendlist)
		}
		send, _ := json.Marshal(al)
		w.Write(send)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}
/*
リクエストをTaskqueueに転送してハンドリングするパッケージです。
*/
package taskqueue

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"google.golang.org/appengine"
	aelog "google.golang.org/appengine/log"
	"google.golang.org/appengine/taskqueue"
)

/*
forwardHandlerとtaskHandlerをhandleFuncを使ってセットします。


URL

各URLは以下となります。
	// ルール
	transfer -> path
	task -> /task/{path}

	// 例
	path = /test
	transfer -> /test
	task -> /task/test

ForwardHandler

forwardHandlerは受け取ったリクエストをtaskHandlerに転送します。
転送する内容はbodyとクエリパラメータの2つです。

TaskHandler

taskHandlerは、handlerへのリクエストをtaskHandlerに転送し、taskQueueを使ってハンドリングします。

*/
func HandleFuncs(path, queueName string, handler http.HandlerFunc) {
	http.HandleFunc(path, forwardHandler(path, queueName))
	http.HandleFunc(fmt.Sprintf("/task%s", path), handler)
}

/*
forwardHandlerを生成します。
forwardHandlerは受け取ったリクエストをtaskHandlerに転送します。
転送する内容はbodyとクエリパラメータの2つです。
*/
func forwardHandler(path, queueName string) http.HandlerFunc {
	f := func(w http.ResponseWriter, r *http.Request) {
		ctx := appengine.NewContext(r)

		// body取得
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			errStr := errors.Wrap(err, "body取得").Error()
			aelog.Errorf(ctx, errStr)
			http.Error(w, errStr, 500)
		}
		defer r.Body.Close()

		u, _ := url.Parse(fmt.Sprintf("/task%s", path))
		u.RawQuery = r.URL.Query().Encode()

		// task作成
		task := &taskqueue.Task{
			Path:    u.RequestURI(),
			Payload: body,
			Header:  r.Header,
			Method:  "POST",
		}
		_, err = taskqueue.Add(ctx, task, queueName)
		if err != nil {
			errStr := errors.Wrap(err, "task作成").Error()
			aelog.Errorf(ctx, errStr)
			http.Error(w, errStr, 500)
		}

		w.WriteHeader(http.StatusOK)
	}
	return http.HandlerFunc(f)
}

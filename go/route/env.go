package route

import (
	"itflow/app/handle"
	"itflow/midware"

	"github.com/hyahm/xmux"
)

var Env *xmux.GroupRoute

func init() {
	Env = xmux.NewGroupRoute()
	Env.Pattern("/env/list").Post(handle.EnvList)
	Env.Pattern("/env/add").Get(handle.AddEnv).End(midware.EndLog)
	Env.Pattern("/env/update").Post(handle.UpdateEnv).End(midware.EndLog)
	Env.Pattern("/env/delete").Get(handle.DeleteEnv).End(midware.EndLog)
}

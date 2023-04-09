package main

import (
	"errors"
	"testing"
)

func TestIteration3B(t *testing.T) {
	knownFrameworks := []string{
		"aahframework.org",
		"confetti-framework.com",
		"github.com/abahmed/gearbox",
		"github.com/aerogo/aero",
		"github.com/aisk/vox",
		"github.com/ant0ine/go-json-rest",
		"github.com/aofei/air",
		"github.com/appist/appy",
		"github.com/astaxie/beego",
		"github.com/beatlabs/patron",
		"github.com/bnkamalesh/webgo",
		"github.com/buaazp/fasthttprouter",
		"github.com/claygod/Bxog",
		"github.com/claygod/microservice",
		"github.com/dimfeld/httptreemux",
		"github.com/dinever/golf",
		"github.com/fulldump/golax",
		"github.com/gernest/alien",
		"github.com/gernest/utron",
		"github.com/gin-gonic/gin",
		"github.com/go-chi/chi",
		"github.com/go-goyave/goyave",
		"github.com/go-macaron/macaron",
		"github.com/go-ozzo/ozzo-routing",
		"github.com/go-playground/lars",
		"github.com/go-playground/pure",
		"github.com/go-zoo/bone",
		"github.com/goa-go/goa",
		"github.com/goadesign/goa",
		"github.com/goanywhere/rex",
		"github.com/gocraft/web",
		"github.com/gofiber/fiber",
		"github.com/goji/goji",
		"github.com/gookit/rux",
		"github.com/gorilla/mux",
		"github.com/goroute/route",
		"github.com/gotuna/gotuna",
		"github.com/gowww/router",
		"github.com/GuilhermeCaruso/bellt",
		"github.com/hidevopsio/hiboot",
		"github.com/husobee/vestigo",
		"github.com/i-love-flamingo/flamingo",
		"github.com/i-love-flamingo/flamingo-commerce",
		"github.com/ivpusic/neo",
		"github.com/julienschmidt/httprouter",
		"github.com/labstack/echo",
		"github.com/lunny/tango",
		"github.com/mustafaakin/gongular",
		"github.com/nbari/violetear",
		"github.com/nsheremet/banjo",
		"github.com/NYTimes/gizmo",
		"github.com/paulbellamy/mango",
		"github.com/rainycape/gondola",
		"github.com/razonyang/fastrouter",
		"github.com/rcrowley/go-tigertonic",
		"github.com/resoursea/api",
		"github.com/revel/revel",
		"github.com/rs/xmux",
		"github.com/twharmon/goweb",
		"github.com/uadmin/uadmin",
		"github.com/ungerik/go-rest",
		"github.com/vardius/gorouter",
		"github.com/VividCortex/siesta",
		"github.com/xujiajun/gorouter",
		"github.com/xxjwxc/ginrpc",
		"github.com/yarf-framework/yarf",
		"github.com/zpatrick/fireball",
		"gobuffalo.io",
		"rest-layer.io",
	}

	e := New(t)
	serverSourcePath := ServerSourcePath(e)
	err := usesKnownPackage(t, serverSourcePath, knownFrameworks)

	if errors.Is(err, errUsageFound) {
		return
	}
	if err == nil || errors.Is(err, errUsageNotFound) {
		e.Errorf("Не найдено использование хотя бы одного известного HTTP фреймворка по пути %q", serverSourcePath)
		return
	}
	e.Errorf("Неожиданная ошибка при поиске использования фреймворка по пути %q, %v", serverSourcePath, err)
}

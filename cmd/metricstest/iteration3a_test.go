package main

import (
	"errors"

	"github.com/stretchr/testify/suite"
)

type Iteration3ASuite struct {
	suite.Suite

	knownFrameworks      PackageRules
	restrictedFrameworks PackageRules
}

// SetupSuite подготавливает необходимые зависимости
func (suite *Iteration3ASuite) SetupSuite() {
	// проверяем наличие необходимых флагов
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")

	// список известных фреймворков
	suite.knownFrameworks = PackageRules{
		{Name: "aahframework.org"},
		{Name: "confetti-framework.com"},
		{Name: "github.com/abahmed/gearbox"},
		{Name: "github.com/aerogo/aero"},
		{Name: "github.com/aisk/vox"},
		{Name: "github.com/ant0ine/go-json-rest"},
		{Name: "github.com/aofei/air"},
		{Name: "github.com/appist/appy"},
		{Name: "github.com/astaxie/beego"},
		{Name: "github.com/beatlabs/patron"},
		{Name: "github.com/bnkamalesh/webgo"},
		{Name: "github.com/claygod/Bxog"},
		{Name: "github.com/claygod/microservice"},
		{Name: "github.com/dimfeld/httptreemux"},
		{Name: "github.com/dinever/golf"},
		{Name: "github.com/fulldump/golax"},
		{Name: "github.com/gernest/alien"},
		{Name: "github.com/gernest/utron"},
		{Name: "github.com/gin-gonic/gin"},
		{Name: "github.com/go-chi/chi"},
		{Name: "github.com/go-goyave/goyave"},
		{Name: "github.com/go-macaron/macaron"},
		{Name: "github.com/go-ozzo/ozzo-routing"},
		{Name: "github.com/go-playground/lars"},
		{Name: "github.com/go-playground/pure"},
		{Name: "github.com/go-zoo/bone"},
		{Name: "github.com/goa-go/goa"},
		{Name: "github.com/goadesign/goa"},
		{Name: "github.com/goanywhere/rex"},
		{Name: "github.com/gocraft/web"},
		{Name: "github.com/gofiber/fiber"},
		{Name: "github.com/goji/goji"},
		{Name: "github.com/gookit/rux"},
		{Name: "github.com/gorilla/mux"},
		{Name: "github.com/goroute/route"},
		{Name: "github.com/gotuna/gotuna"},
		{Name: "github.com/gowww/router"},
		{Name: "github.com/GuilhermeCaruso/bellt"},
		{Name: "github.com/hidevopsio/hiboot"},
		{Name: "github.com/husobee/vestigo"},
		{Name: "github.com/i-love-flamingo/flamingo"},
		{Name: "github.com/i-love-flamingo/flamingo-commerce"},
		{Name: "github.com/ivpusic/neo"},
		{Name: "github.com/julienschmidt/httprouter"},
		{Name: "github.com/labstack/echo"},
		{Name: "github.com/lunny/tango"},
		{Name: "github.com/mustafaakin/gongular"},
		{Name: "github.com/nbari/violetear"},
		{Name: "github.com/nsheremet/banjo"},
		{Name: "github.com/NYTimes/gizmo"},
		{Name: "github.com/paulbellamy/mango"},
		{Name: "github.com/rainycape/gondola"},
		{Name: "github.com/razonyang/fastrouter"},
		{Name: "github.com/rcrowley/go-tigertonic"},
		{Name: "github.com/resoursea/api"},
		{Name: "github.com/revel/revel"},
		{Name: "github.com/rs/xmux"},
		{Name: "github.com/twharmon/goweb"},
		{Name: "github.com/uadmin/uadmin"},
		{Name: "github.com/ungerik/go-rest"},
		{Name: "github.com/vardius/gorouter"},
		{Name: "github.com/VividCortex/siesta"},
		{Name: "github.com/xujiajun/gorouter"},
		{Name: "github.com/xxjwxc/ginrpc"},
		{Name: "github.com/yarf-framework/yarf"},
		{Name: "github.com/zpatrick/fireball"},
		{Name: "gobuffalo.io"},
		{Name: "rest-layer.io"},
	}

	// список запрещенных фреймворков
	suite.restrictedFrameworks = PackageRules{
		{Name: "github.com/valyala/fasthttp"},
		{Name: "github.com/fasthttp/router"},
	}
}

// TestFrameworkUsage пробует рекурсивно найти хотя бы одно использование известных фреймворков в директории с исходным кодом проекта
func (suite *Iteration3ASuite) TestFrameworkUsage() {
	// проверяем наличие запрещенных фреймворков
	err := usesKnownPackage(suite.T(), flagTargetSourcePath, suite.restrictedFrameworks...)
	if errors.Is(err, errUsageFound) {
		suite.T().Errorf("Найдено использование одного из не рекомендуемых фреймворков по пути %s: %s",
			flagTargetSourcePath, suite.restrictedFrameworks.PackageList())
		return
	}

	// проверяем наличие известных фреймворков
	err = usesKnownPackage(suite.T(), flagTargetSourcePath, suite.knownFrameworks...)
	if errors.Is(err, errUsageFound) {
		return
	}
	if errors.Is(err, errUsageNotFound) {
		suite.T().Errorf("Не найдено использование хотя бы одного известного HTTP фреймворка по пути %s", flagTargetSourcePath)
		return
	}
	suite.T().Errorf("Неожиданная ошибка при поиске использования фреймворка по пути %s: %s", flagTargetSourcePath, err)
}

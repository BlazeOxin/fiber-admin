package admin

import (
	"reflect"

	"github.com/gofiber/fiber"
)

type model struct {
	name   string
	object interface{}
}

var sectionMap = make(map[string][]model)

func getStructName(structure interface{}) string {
	valueOf := reflect.ValueOf(structure)

	if valueOf.Type().Kind() == reflect.Ptr {
		return reflect.Indirect(valueOf).Type().Name()
	}
	return valueOf.Type().Name()
}

/*AddSection :
register your Database Structs into different sections
effictively mimic Django Admin App based Object Grouping
*/
func AddSection(name string, inputStructs ...interface{}) {
	var sectionStructs []model
	for _, iterStruct := range inputStructs {
		sectionStructs = append(sectionStructs, model{
			name:   getStructName(iterStruct),
			object: iterStruct,
		})
	}
	sectionMap[name] = sectionStructs
}

/*SetupRoutes :
function creates all the necessary routes for the admin site
*/
func SetupRoutes(app *fiber.App) {

	app.Get("/admin/", func(c *fiber.Ctx) {
		c.Render("admin/home", fiber.Map{
			"Sections": sectionMap,
		}, "layouts/admin")
	})
}

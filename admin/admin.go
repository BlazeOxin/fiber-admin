package admin

import (
	"fmt"
	"reflect"

	"github.com/fatih/structs"
	"github.com/gofiber/fiber"
	"github.com/gosimple/slug"
	"gorm.io/gorm"
)

/*Model :

 */
type Model struct {
	Name   string
	Object interface{}
}

/*ContentManager :

 */
var ContentManager = make(map[string][]Model)

func getStructName(structure interface{}) string {
	valueOf := reflect.ValueOf(structure)

	if valueOf.Type().Kind() == reflect.Ptr {
		return reflect.Indirect(valueOf).Type().Name()
	}
	return valueOf.Type().Name()
}

func getStructNameList(structList []interface{}) []string {
	var output []string
	for _, v := range structList {
		output = append(output, getStructName(v))
	}
	return output
}

/*AddSection :
register your Database Structs into different sections
effictively mimic Django Admin App based Object Grouping
*/
func AddSection(name string, inputStructs ...interface{}) {
	var sectionStructs []Model
	for _, iterStruct := range inputStructs {

		sectionStructs = append(sectionStructs, Model{
			Name:   getStructName(iterStruct),
			Object: iterStruct,
		})
	}
	ContentManager[name] = sectionStructs
}

/*SetupRoutes :
function creates all the necessary routes for the admin site
*/
func SetupRoutes(app *fiber.App, db *gorm.DB) {
	app.Get("/admin", func(c *fiber.Ctx) {
		c.Render("admin/home", fiber.Map{
			"ContentManager":        ContentManager,
			"Slugify":               slug.Make,
			"ShowContentManagement": len(ContentManager) > 0,
		}, "layout/admin")
	})

	for sectionName, contentList := range ContentManager {
		app.Get(fmt.Sprintf("/admin/%s", slug.Make(sectionName)), func(c *fiber.Ctx) {
			c.Render("admin/template/section", fiber.Map{
				"SectionName": sectionName,
				"StructList":  contentList,
				"Slugify":     slug.Make,
			}, "layout/admin")
		})

		for _, contentStruct := range contentList {
			var queriedData []interface{}
			db.Model(contentStruct.Object).Find(&queriedData)
			fmt.Print(queriedData)

			app.Get(fmt.Sprintf("/admin/%s/%s", slug.Make(sectionName), slug.Make(contentStruct.Name)), func(c *fiber.Ctx) {
				c.Render("admin/template/content", fiber.Map{
					"ContentName": contentStruct.Name,
					"SectionName": sectionName,
					"Fields":      structs.Names(contentStruct.Object),
					"Items":       getStructNameList(queriedData),
					"getValue": func(fieldName string, index int) interface{} {
						return reflect.ValueOf(queriedData[index]).FieldByName(fieldName)
					},
					"Slugify": slug.Make,
				}, "layout/admin")
			})

		}
	}
}

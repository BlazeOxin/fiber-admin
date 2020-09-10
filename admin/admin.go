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

/*Field :

 */
type Field struct {
	Type    string
	Select  bool
	Choices []string
}

/*FieldWithValue :

 */
type FieldWithValue struct {
	Type    string
	Select  bool
	Choices []string
	Value   interface{}
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
func createStructFieldMap(inputStruct interface{}) map[string]interface{} {
	var output map[string]interface{}

	for _, fieldName := range structs.Names(inputStruct) {
		valueOf := reflect.ValueOf(inputStruct)
		field := valueOf.FieldByName(fieldName)

		fieldSelect := false
		fieldType := "text"

		switch field.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint32, reflect.Uint64:
			fieldType = "number"
			break
		case reflect.String:
			fieldType = "text"
			break
		case reflect.Bool:
			fieldType = "checkbox"
			break
		case reflect.Struct:
			fieldType = "foreignkey"
			fieldSelect = true
			break
		}
		output[fieldName] = Field{
			Select: fieldSelect,
			Type:   fieldType,
		}
	}

	return output
}

func createStructFieldWithValueMap(inputStruct interface{}) map[string]interface{} {
	var output map[string]interface{}

	for _, fieldName := range structs.Names(inputStruct) {
		valueOf := reflect.Indirect(reflect.ValueOf(inputStruct))
		field := valueOf.FieldByName(fieldName)

		fieldSelect := false
		fieldType := "text"
		var fieldValue interface{}

		switch field.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint32, reflect.Uint64:
			fieldType = "number"
			fieldValue = field
			break
		case reflect.String:
			fieldType = "text"
			fieldValue = field
			break
		case reflect.Bool:
			fieldType = "checkbox"
			fieldValue = field
			break
		case reflect.Struct:
			fieldValue = createStructFieldWithValueMap(field)
			fieldSelect = true
			fieldType = "foreignkey"
			break
		default:
			fieldValue = field
			break
		}
		output[fieldName] = FieldWithValue{
			Select: fieldSelect,
			Value:  fieldValue,
			Type:   fieldType,
		}
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
			"Title":                 "Admin Site",
			"ContentManager":        ContentManager,
			"Slugify":               slug.Make,
			"ShowContentManagement": len(ContentManager) > 0,
		}, "layout/admin")
	})

	for sectionName, contentList := range ContentManager {
		app.Get(fmt.Sprintf("/admin/%s", slug.Make(sectionName)), func(c *fiber.Ctx) {
			c.Render("admin/template/section", fiber.Map{
				"Title":       sectionName,
				"SectionName": sectionName,
				"StructList":  contentList,
				"Slugify":     slug.Make,
			}, "layout/admin")
		})

		for _, contentStruct := range contentList {
			var queriedData []interface{}
			model := db.Model(contentStruct.Object)
			model.Find(&queriedData)
			fmt.Print(queriedData)

			app.Get(fmt.Sprintf("/admin/%s/%s", slug.Make(sectionName), slug.Make(contentStruct.Name)), func(c *fiber.Ctx) {
				c.Render("admin/template/content", fiber.Map{
					"Title":       contentStruct.Name,
					"ContentName": contentStruct.Name,
					"SectionName": sectionName,
					"Fields":      structs.Names(contentStruct.Object),
					"Items":       getStructNameList(queriedData),
					"getValue": func(index int) interface{} {
						return queriedData[index]
					},
					"Slugify": slug.Make,
				}, "layout/admin")
			})

			app.Get(fmt.Sprintf("/admin/%s/%s/:id", slug.Make(sectionName), slug.Make(contentStruct.Name)), func(c *fiber.Ctx) {
				id := c.Params("id")
				var contentObjectData interface{}

				err := model.First(&contentObjectData, id).Error

				context := fiber.Map{
					"Title":       fmt.Sprintf("Edit %s", contentStruct.Name),
					"ContentName": contentStruct.Name,
					"SectionName": sectionName,
					"Slugify":     slug.Make,
					"Data":        contentObjectData,
					"HasError":    err != nil,
				}
				if err == nil {
					context["ID"] = id
					context["Fields"] = createStructFieldWithValueMap(contentStruct.Object)
				}
				c.Render("admin/template/edit", context, "layout/admin")
			})
			app.Get(fmt.Sprintf("/admin/%s/%s/create", slug.Make(sectionName), slug.Make(contentStruct.Name)), func(c *fiber.Ctx) {
				c.Render("admin/template/create", fiber.Map{
					"Title":       fmt.Sprintf("Create %s", contentStruct.Name),
					"ContentName": contentStruct.Name,
					"SectionName": sectionName,
					"Slugify":     slug.Make,
					"Fields":      structs.Names(contentStruct.Object),
				}, "layout/admin")
			})

		}
	}
}

package admin

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/fatih/structs"
	"github.com/fatih/structtag"
	"github.com/gofiber/fiber"
	"github.com/gosimple/slug"
	"gorm.io/gorm"
)

/*Model :
 */
type Model struct {
	ObjectPtr interface{}
	Fields    map[string]Field
}

/*Field :
 */
type Field struct {
	Type            string
	Choices         []string
	ForeignKeyModel string
}

var lModels = make(map[string]Model)
var lApps = make(map[string][]string)

func getStructName(structure interface{}) string {
	valueOf := reflect.ValueOf(structure)

	if valueOf.Type().Kind() == reflect.Ptr {
		return reflect.Indirect(valueOf).Type().Name()
	}
	return valueOf.Type().Name()
}

func getFieldType(field reflect.StructField) string {
	fType := strings.Replace(fmt.Sprintf("%s", field.Type), "[]", "", -1)
	if strings.Contains(fType, ".") {
		return strings.Split(fType, ".")[1]
	}
	return fType
}

func getStructNameList(structList []interface{}) []string {
	var output []string
	for _, v := range structList {
		output = append(output, getStructName(v))
	}
	return output
}

func processFields(Model interface{}) map[string]Field {
	Output := make(map[string]Field)
	ModelValue := reflect.Indirect(reflect.ValueOf(Model))

	for _, fieldName := range structs.Names(Model) {
		fieldByValue := ModelValue.FieldByName(fieldName)
		fieldByType, _ := ModelValue.Type().FieldByName(fieldName)

		var fieldType string
		var foreignKeyModel string

		tags, _ := structtag.Parse(string(fieldByType.Tag))

		gormTag, _ := tags.Get("gorm")
		if gormTag != nil {
			switch gormTag.Name {
			case "primaryKey":
				fieldType = "primarykey"
			case "ForeignKey":
				fieldType = "foreginkey"
			}
			if gormTag.Name == "primaryKey" {
				fieldType = "primarykey"
			} else if strings.Contains(gormTag.Name, "ForeignKey") {
				fieldType = "foreignkey"
				foreignKeyModel = getFieldType(fieldByType)
			}
		}

		adminTag, _ := tags.Get("admin")
		if adminTag != nil {
			switch adminTag.Name {
			case "textarea":
				fieldType = "textarea"
			}
		}

		if fieldType == "" {
			switch fieldByValue.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
				fieldType = "number"
			case reflect.String:
				fieldType = "text"
			case reflect.Array:
				fieldType = "manytomany"
				foreignKeyModel = getFieldType(fieldByType)
			case reflect.Bool:
				fieldType = "checkbox"
			}
		}

		Output[fieldName] = Field{
			Type:            fieldType,
			ForeignKeyModel: foreignKeyModel,
		}
	}

	return Output
}

/*AddSection :
register your Database Structs into different sections
effictively mimic Django Admin App based Object Grouping
*/
func AddSection(name string, inputStructs ...interface{}) {
	ModelList := []string{}
	for _, iterStruct := range inputStructs {
		modelName := getStructName(iterStruct)
		lModels[modelName] = Model{
			ObjectPtr: iterStruct,
			Fields:    processFields(iterStruct),
		}
		ModelList = append(ModelList, modelName)
	}
	lApps[name] = ModelList
	fmt.Println(lApps)
	fmt.Println(lModels)
}

/*SetupRoutes :
function creates all the necessary routes for the admin site
*/
func SetupRoutes(app *fiber.App, db *gorm.DB) {
	app.Get("/admin", func(c *fiber.Ctx) {
		c.Render("admin/home", fiber.Map{
			"Title":                 "Admin Site",
			"Apps":                  lApps,
			"Slugify":               slug.Make,
			"ShowContentManagement": len(lApps) > 0,
		}, "layout/admin")
	})

	for AppName, ContentList := range lApps {
		app.Get(fmt.Sprintf("/admin/%s", slug.Make(AppName)), func(c *fiber.Ctx) {
			c.Render("admin/template/section", fiber.Map{
				"Title":      AppName,
				"AppName":    AppName,
				"StructList": ContentList,
				"Slugify":    slug.Make,
			}, "layout/admin")
		})

		for _, ModelName := range ContentList {
			Model := lModels[ModelName]

			var queriedData []interface{}
			DBModel := db.Model(Model.ObjectPtr)
			DBModel.Find(&queriedData)
			fmt.Print(queriedData)

			app.Get(fmt.Sprintf("/admin/%s/%s", slug.Make(AppName), slug.Make(ModelName)), func(c *fiber.Ctx) {
				c.Render("admin/template/content", fiber.Map{
					"Title":     ModelName,
					"AppName":   AppName,
					"ModelName": ModelName,
					"Fields":    lModels[ModelName],
					"Slugify":   slug.Make,
				}, "layout/admin")
			})

			app.Get(fmt.Sprintf("/admin/%s/%s/:id", slug.Make(AppName), slug.Make(ModelName)), func(c *fiber.Ctx) {
				id := c.Params("id")
				err := DBModel.First(lModels[ModelName].ObjectPtr, id).Error
				context := fiber.Map{
					"Title":     fmt.Sprintf("Edit %s", ModelName),
					"AppName":   AppName,
					"ModelName": ModelName,
					"Slugify":   slug.Make,
					"Fields":    lModels[ModelName],
					"HasError":  err != nil,
				}
				if err == nil {
					context["ID"] = id
				}
				c.Render("admin/template/edit", context, "layout/admin")
			})
			app.Get(fmt.Sprintf("/admin/%s/%s/create", slug.Make(AppName), slug.Make(ModelName)), func(c *fiber.Ctx) {
				c.Render("admin/template/create", fiber.Map{
					"Title":     fmt.Sprintf("Create %s", ModelName),
					"AppName":   AppName,
					"ModelName": ModelName,
					"Slugify":   slug.Make,
					"Fields":    lModels[ModelName],
				}, "layout/admin")
			})

		}
	}
}

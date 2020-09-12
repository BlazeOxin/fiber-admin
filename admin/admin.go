package admin

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/fatih/structs"
	"github.com/fatih/structtag"
	"github.com/gofiber/fiber"
	"github.com/gosimple/slug"
	"gorm.io/gorm"
)

/*MConfig :

 */
type MConfig struct {
	ListDisplay []string
}

/*Model :

 */
type Model struct {
	ObjectPtr       interface{}
	Fields          map[string]Field
	PrimaryKeyField string
	Config          MConfig
}

/*Field :

 */
type Field struct {
	Type            string
	Choices         []string
	HelpText        string
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

func processFields(Model interface{}) (map[string]Field, string) {
	Output := make(map[string]Field)
	ModelValue := reflect.Indirect(reflect.ValueOf(Model))
	var primaryKeyField string
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
				primaryKeyField = fieldName
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

	return Output, primaryKeyField
}

func parseFieldType(input string, field Field) interface{} {
	var Output interface{}

	switch field.Type {

	case "number", "foreginkey", "primarykey":
		Output, _ = strconv.ParseInt(input, 10, 0)
	case "checkbox":
		Output, _ = strconv.ParseBool(input)
	case "manytomany":
		manyToMany := []int64{}
		for _, value := range strings.Split(input, ",") {
			converted, _ := strconv.ParseInt(value, 10, 0)
			manyToMany = append(manyToMany, converted)
		}
		Output = manyToMany
	}

	return Output
}

func getPOSTData(body string, fields map[string]Field) map[string]interface{} {
	Output := make(map[string]interface{})

	for _, term := range strings.Split(body, "&") {
		SplitTerm := strings.Split(term, "=")
		Name := SplitTerm[0]
		SValue, _ := url.QueryUnescape(SplitTerm[1])
		Key := strings.ToLower(Name)

		// parseFieldType(
		Output[Key] = SValue
		// , fields[SplitTerm[0]])
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

		Fields, PrimaryKeyField := processFields(iterStruct)

		lModels[modelName] = Model{
			ObjectPtr:       iterStruct,
			Fields:          Fields,
			PrimaryKeyField: PrimaryKeyField,
		}
		ModelList = append(ModelList, modelName)
	}
	lApps[name] = ModelList
}

/*ConfigCMSModel :

 */
func ConfigCMSModel(ModelName string, Config *MConfig) {
	newModelConfig := lModels[ModelName]
	newModelConfig.Config = *Config
	lModels[ModelName] = newModelConfig
}

/*SetupRoutes :
function creates all the necessary routes for the admin site
*/
func SetupRoutes(app *fiber.App, db *gorm.DB) {
	app.Get("/admin", func(c *fiber.Ctx) {
		c.Render("admin/home", fiber.Map{
			"Title":   "Admin Site",
			"Apps":    lApps,
			"Slugify": slug.Make,
			"ShowCMS": len(lApps) > 0,
		}, "layout/admin")
	})

	for AppName, ModelList := range lApps {
		AppSlug := slug.Make(AppName)
		app.Get(fmt.Sprintf("/admin/%s", AppSlug), func(c *fiber.Ctx) {
			c.Render("admin/app", fiber.Map{
				"Title":     AppName,
				"AppName":   AppName,
				"ModelList": ModelList,
				"Slugify":   slug.Make,
			}, "layout/admin")
		})
		for i := range ModelList {
			ModelName := ModelList[i]
			ModelSlug := slug.Make(ModelName)
			Model := lModels[ModelName]

			app.Get(fmt.Sprintf("/admin/%s/%s", AppSlug, ModelSlug), func(c *fiber.Ctx) {
				var queriedData []map[string]interface{}
				db.Model(Model.ObjectPtr).Find(&queriedData)

				c.Render("admin/cms/model", map[string]interface{}{
					"Title":     ModelName,
					"AppName":   AppName,
					"ModelName": ModelName,
					"Fields":    Model,
					"Data":      queriedData,
					"getPrimaryKey": func(index int) interface{} {
						return queriedData[index][strings.ToLower(Model.PrimaryKeyField)]
					},
					"getData": func(index int, fieldName string) interface{} {
						return queriedData[index][strings.ToLower(fieldName)]
					},
					"ListDisplay": Model.Config.ListDisplay,
					"isEmpty":     len(Model.Config.ListDisplay) == 0,
					"Slugify":     slug.Make,
				}, "layout/admin")

			})
			app.Post(fmt.Sprintf("/admin/%s/%s", AppSlug, ModelSlug), func(c *fiber.Ctx) {
				postData := getPOSTData(c.Body(), Model.Fields)

				db.Model(Model.ObjectPtr).Create(postData)

				var queriedData []map[string]interface{}
				db.Model(Model.ObjectPtr).Find(&queriedData)
				c.Render("admin/cms/model", map[string]interface{}{
					"Title":     ModelName,
					"AppName":   AppName,
					"ModelName": ModelName,
					"Fields":    Model,
					"Data":      queriedData,
					"getPrimaryKey": func(index int) interface{} {
						return queriedData[index][strings.ToLower(Model.PrimaryKeyField)]
					},
					"getData": func(index int, fieldName string) interface{} {
						return queriedData[index][strings.ToLower(fieldName)]
					},
					"ListDisplay": Model.Config.ListDisplay,
					"isEmpty":     len(Model.Config.ListDisplay) == 0,
					"Slugify":     slug.Make,
				}, "layout/admin")
			})
			app.Delete(fmt.Sprintf("/admin/%s/%s", AppSlug, ModelSlug), func(c *fiber.Ctx) {
				postData := getPOSTData(c.Body(), Model.Fields)
				var ToDelete []int
				for _, v := range strings.Split(fmt.Sprintf("%s", postData["objects"]), ",") {
					iValue, _ := strconv.ParseInt(v, 10, 32)
					ToDelete = append(ToDelete, int(iValue))
				}
				fmt.Println(postData)
				db.Delete(Model.ObjectPtr, ToDelete)
				c.Send("")
			})
			app.Get(fmt.Sprintf("/admin/%s/%s/edit/:id", AppSlug, ModelSlug), func(c *fiber.Ctx) {
				id := c.Params("id")

				queriedData := make(map[string]interface{})
				err := db.Model(Model.ObjectPtr).First(queriedData, id).Error

				context := map[string]interface{}{
					"Title":     fmt.Sprintf("Edit %s", ModelName),
					"AppName":   AppName,
					"ModelName": ModelName,
					"getData": func(fieldName string) interface{} {
						return queriedData[strings.ToLower(fieldName)]
					},
					"PrimaryKey": queriedData[strings.ToLower(Model.PrimaryKeyField)],
					"Slugify":    slug.Make,
					"Fields":     Model.Fields,
					"HasError":   err != nil,
				}

				if err == nil {
					context["ID"] = id
				} else {
					fmt.Println(err)
				}

				c.Render("admin/cms/edit", context, "layout/admin")
			})

			app.Post(fmt.Sprintf("/admin/%s/%s/edit/:id", AppSlug, ModelSlug), func(c *fiber.Ctx) {
				id := c.Params("id")
				fNamePK := strings.ToLower(Model.PrimaryKeyField)
				err := db.Where("? = ?", fNamePK, id).First(Model.ObjectPtr).Error
				if err != nil {
					fmt.Println(err)
				}
				// postdata := getPOSTData(c.Body(), Model.Fields)
				fmt.Println(reflect.Indirect(reflect.ValueOf(Model.ObjectPtr)))
				// fmt.Println(db.Model(Model.ObjectPtr).Where("? = ?", strings.ToLower(Model.PrimaryKeyField), id).Updates(postdata).Error)

				c.Send("queriedData")
			})

			app.Get(fmt.Sprintf("/admin/%s/%s/create", AppSlug, ModelSlug), func(c *fiber.Ctx) {
				c.Render("admin/cms/create", fiber.Map{
					"Title":     fmt.Sprintf("Create %s", ModelName),
					"AppName":   AppName,
					"ModelName": ModelName,
					"Slugify":   slug.Make,
					"Fields":    Model.Fields,
				}, "layout/admin")
			})

		}
	}
}

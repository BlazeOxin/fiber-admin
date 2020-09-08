package admin

import (
	"github.com/gofiber/fiber"
)

/*CreateAdminSite :
function places the dynamic content for the site
*/
func CreateAdminSite() {

}

/*SetAdminRoutes :
function creates all the necessary routes for the admin site
*/
func SetAdminRoutes(app *fiber.App) {
	app.Get("/admin/", func(c *fiber.Ctx) {
		c.Render("name", fiber.Map{}, "layouts/admin")
	})
}

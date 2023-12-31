package controllers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/FrancoRutigliano/EcommerceGolang/database"
	"github.com/FrancoRutigliano/EcommerceGolang/models"
	generate "github.com/FrancoRutigliano/EcommerceGolang/tokens"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Declaración e inicialización de la variable UserCollection que apunta a una colección de usuarios en MongoDB.
var UserCollection *mongo.Collection = database.UserData(database.Client, "Users")

// Declaración e inicialización de la variable ProductCollection que apunta a una colección de productos en MongoDB.
var ProductCollection *mongo.Collection = database.ProductData(database.Client, "Products")

// Declaración e inicialización de la variable Validate como un validador nuevo esta variable Validate es
// una instancia de un validador que se utilizará para validar datos en el código.
var Validate = validator.New()

func HashPassword(password string) string {
	panic("Not Used Yet")
}

func VerifyPassword(userPassword string, givePassword string) (bool, string) {
	panic("Not Used Yet")
}

func Sigup() gin.HandlerFunc {
	// Esta funcion maneja el registro de los usuarios
	return func(c *gin.Context) {
		// Se crea un contexto con un timeOut de 100 segundos
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel() // Siempre nos aseguraremos de que el contexto finalice al terminar la función

		// Se crea una variable user del módelo 'User' para almacenar los datos del usuario
		var user models.User
		// Intentaremos extraer y parsear los datos del JSON del cuerpo de la solicitud al módelo user.
		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"erorr": err.Error()})
			return
		}
		// Se valida la estructura del usuario usando
		/*
			El objetivo principal de esta sección es garantizar que los datos proporcionados en
			la solicitud cumplan con los criterios definidos para la estructura User.
			Esto ayuda a asegurar la integridad y consistencia de los datos antes de continuar
			con el proceso de registro del usuario.
		*/
		validationErr := Validate.Struct(user)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr})
			return
		}
		// Se verifica si el correo electronico ya esta en la base de datos
		count, err := UserCollection.CountDocuments(ctx, bson.M{"email": user.Email})
		if err != nil {
			log.Panic(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}

		if count > 0 {
			// Si el correo electronico ya existe, se devuelve un error
			c.JSON(http.StatusBadRequest, gin.H{"error": "user email already exist"})
		}

		// Vereficamos si el numero de telefono del usuario ya existe en la base de datos.
		count, err = UserCollection.CountDocuments(ctx, bson.M{"phone": user.Phone})

		defer cancel()

		if err != nil {
			log.Panic(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}

		if count > 0 {
			// Si el numero de telefono ya esta en uso se devuelve un error.
			c.JSON(http.StatusBadRequest, gin.H{"error": "this phone no. is already in use"})
			return
		}
		// HashPassword convierte la contraseña en una
		// cadena irreversible para protegerla en la base de datos.
		password := HashPassword(*user.Password)
		// En vez de guardar la contraseña en texto(String), la guardamos en la base de datos hasheada
		user.Password = &password

		// Se establecen las fechas de creación y actualización del usuario
		user.Created_At, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		user.Updated_At, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		user.ID = primitive.NewObjectID()
		user.User_ID = user.ID.Hex()

		// Se generan tokens de autenticación para el usuario
		token, refreshtoken, _ := generate.TokenGenerator(*user.Email, *user.First_Name, *user.Last_Name, user.User_ID)
		user.Token = &token
		user.Refresh_Token = &refreshtoken

		// Se inicializan las listas asociadas al usuario (carrito, direcciones, estado de órdenes)
		user.UserCart = make([]models.ProductUser, 0)
		user.Address_Details = make([]models.Address, 0)
		user.Order_Status = make([]models.Order, 0)

		// Se inserta el usuario en la base de datos
		/*
			Utilizamos UserCollection.InsertOne para guardar el objeto user en la base de datos.
			Si ocurre algún error durante el proceso de inserción, se envía un mensaje de
			error al cliente indicando que la creación del usuario no se completó correctamente.
		*/
		_, inserterr := UserCollection.InsertOne(ctx, user)
		if inserterr != nil {
			// Si hay un error al insertar el usuario, se devuelve un error
			c.JSON(http.StatusInternalServerError, gin.H{"error": "the user did not get created"})
			return
		}
		defer cancel()

		// Si todo salió bien, se devuelve un mensaje de exito al cliente.
		c.JSON(http.StatusCreated, "Successfully Signed In")

	}
}

func Login() gin.HandlerFunc {

	return func(c *gin.Context) {
		// Crear un contexto con un límite de tiempo de 100 segundos
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel() // Cancelar el contexto cuando la función retorne

		var user models.User      // Crear una variable para almacenar un usuario
		var founduser models.User // Crear una variable para almacenar un usuario encontrado

		// Intentar vincular el cuerpo de la solicitud JSON al objeto 'user'
		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err}) // Enviar una respuesta de error si hay un problema con el JSON
		}

		// Buscar un usuario en la base de datos usando el email proporcionado en 'user'
		err := UserCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&founduser)
		defer cancel() // Asegurarse de cancelar el contexto al final de la función

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "login or password incorrect"})
			return // Enviar un mensaje de error si no se encuentra el usuario en la base de datos
		}

		// ... (Aquí faltaría agregar la lógica para comparar contraseñas o realizar alguna acción con el usuario encontrado)
		PasswordIsValid, msg := VerifyPassword(*user.Password, *founduser.Password)

		defer cancel()

		// Todo esta lógica estaría sucediendo si la contraseña no es valida.
		// Para determinar esto, tenemos que checkear la password de ese usuario que tenemos en la DB y las Password que el usuario nos entrega en el login
		if !PasswordIsValid {
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			fmt.Println(msg)
			return
		}
		// Si estas dos contraseñas "machean", generamos el token
		token, refreshToken, _ := generate.TokenGenerator(*founduser.Email, *founduser.First_Name, *founduser.Last_Name, founduser.User_ID)
		defer cancel()
		// luego de generar el token, vamos a actualizar todos los tokens.
		// le pasaremos el token y el token y el id de usuario
		generate.UpdateAllTokens(token, refreshToken, founduser.User_ID)

		// Caso de que todo funcione bien, devolvemos un estado http de encontrado y el usuario encontrado
		c.JSON(http.StatusFound, founduser)

	}

}

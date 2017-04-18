package main

import (
	//"github.com/gorilla/mux"
	"net/http"
	"html/template"
	_ "github.com/go-sql-driver/mysql"
	"github.com/urfave/negroni"
	"github.com/goincremental/negroni-sessions"
	"github.com/goincremental/negroni-sessions/cookiestore"
	gmux "github.com/gorilla/mux"   //gmux is a reference name to gorilla mux library
	"os"
	"log"
	"fmt"
	"strings"
	"bufio"
	"golang.org/x/crypto/bcrypt"
	"database/sql"
	"encoding/json"

)

type DBHandler struct {
	db *sql.DB //removed declaration here to easily access the variable from our middleware handler

}

type User struct {
	UserName string `db:"username"`
	Secret []byte `db:"secret"`
}
type PContacts struct {
	Cont []Contacts
}
type Contacts struct {
	Name string `db:"name"`
	Mob string `db:"mob"`
	Email string `db:"email"`
	Address string `db:"address"`
	Nationality string `db:"nationality"`
	UserName string `db:"user_Name"`
}
type OneContact struct {
	Name string `db:"name"`
	Mob string `db:"contact_numbers"`
}
type LoginPage struct {
	Error string
}


type appHandler struct {
	*DBHandler
	H func(http.ResponseWriter, *http.Request, *DBHandler)
}
func (obj appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Updated to pass obj.DBHandler as a parameter to our handler type.
	obj.H( w , r , obj.DBHandler)

}

func serveWeb(){
	dataBase,_ := sql.Open("mysql", "root:20121993@/addressBook")
	context := &DBHandler{db: dataBase }
	mux := gmux.NewRouter()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var objContact Contacts
		//var urlParameter map[string]string
		objContact.UserName = sessions.GetSession(r).Get("username").(string)
		urlParameter := gmux.Vars(r)
		page_alias, staticPage := objContact.serveContent(urlParameter)
		P, err := objContact.serveContentTwo(context)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if page_alias == "home" {
			staticPage.Execute(w,P)
		}else{
			staticPage.Execute(w,nil)
		}


	})
	mux.HandleFunc("/{page_alias}", func(w http.ResponseWriter, r *http.Request) {
		var objContact Contacts
		//var urlParameter map[string]string
		objContact.UserName = sessions.GetSession(r).Get("username").(string)
		urlParameter := gmux.Vars(r)
		page_alias, staticPage := objContact.serveContent(urlParameter)
		P, err := objContact.serveContentTwo(context)
		if err != nil{
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if page_alias == "home" {
			staticPage.Execute(w,P)
		}else{
			staticPage.Execute(w,nil)
		}
	}).Methods("GET")

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		var p LoginPage
		var objUser User
		var regChoice = r.FormValue("register")
		objUser.UserName = r.FormValue("username")
		objUser.Secret = []byte(r.FormValue("password"))
		if regChoice != "" {
			err := objUser.serveRegister(context)
			if err != nil{
				http.Error(w,err.Error(),http.StatusInternalServerError)
				return
			}
			http.Redirect(w,r,"/login",http.StatusFound)
			return
		} else { //login process
			var err error
			var user *sql.Stmt
			var pass User
			err, user, pass = objUser.serveLogin(context)

			//////////////////////////////////////////////////
			if err!= nil {
				fmt.Println("ERROR LOGIN #1: ", err.Error())
				p.Error = err.Error()

			}else if user == nil {
				fmt.Println("ERROR LOGIN #: ", err.Error())
				p.Error = "NO SUCH USER WITH USERNAME "+ objUser.UserName
				return
			}else{//the user isn't nil we'll perform a hard cast on object returned from the database to access user object

				//RETURN NIL ON SUCCESS authentication AND ERROR ON failure
				if err = bcrypt.CompareHashAndPassword([]byte(pass.Secret),[]byte(objUser.Secret)); err != nil {
					//if the err isn't nil let's set the err property on the page object
					fmt.Println("ERROR LOGIN #3: ",err.Error())
					p.Error = err.Error()
					return
				}else {
					sessions.GetSession(r).Set("username",objUser.UserName)
					fmt.Println("I REDIRECTED HERE IN / AGAIN")
					http.Redirect(w,r,"/home",http.StatusFound)
					return
				}
			}
		}

	}).Methods("POST")
	mux.HandleFunc("/add", func(w http.ResponseWriter, r *http.Request) {
		var objAddContact Contacts
		objAddContact.Name =r.FormValue("name")
		objAddContact.Mob = r.FormValue("mob")
		objAddContact.Email = r.FormValue("email")
		objAddContact.Address = r.FormValue("address")
		objAddContact.Nationality = r.FormValue("nationality")
		objAddContact.UserName = sessions.GetSession(r).Get("username").(string)

		b, err := objAddContact.serveAdd(context)
		if err != nil{
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err:= json.NewEncoder(w).Encode(b); err != nil {
			fmt.Println("ERRORRRRR : " + err.Error())
			http.Error(w,err.Error(),http.StatusInternalServerError)
			return
		}
	}).Methods("PUT")

	mux.HandleFunc("/deleteContact/{name}", func(w http.ResponseWriter, r *http.Request) {
		var objDeleteContacts Contacts
		objDeleteContacts.Name = gmux.Vars(r)["name"]
		err := objDeleteContacts.serveDelete(context)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK) //to send 200 status okay along with the response object when we return
	}).Methods("DELETE")

	mux.HandleFunc("/login", logoutHandler).Methods("GET")

	mux.HandleFunc("/home/{name}", func(w http.ResponseWriter, r *http.Request) {
		var objOneContact OneContact
		objOneContact.Name = gmux.Vars(r)["name"]

		P, err := objOneContact.ServeOneContact(context)
		if err != nil{
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err:= json.NewEncoder(w).Encode(P); err != nil {
			fmt.Println("ERRORRRRR One contact: " + err.Error())
			http.Error(w,err.Error(),http.StatusInternalServerError)
			return
		}
	}).Methods("GET")

	mux.HandleFunc("/home/{name}", func(w http.ResponseWriter, r *http.Request) {
		var objAddmoreContactNum OneContact
		objAddmoreContactNum.Name = gmux.Vars(r)["name"]
		objAddmoreContactNum.Mob = r.FormValue("contact_numbers")

		b, err := objAddmoreContactNum.ServeAddMoreContactNum(context)
		if err != nil{
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err:= json.NewEncoder(w).Encode(b); err != nil {
			fmt.Println("ERRORRRRR : " + err.Error())
			http.Error(w,err.Error(),http.StatusInternalServerError)
			return
		}
	}).Methods("PUT")

	mux.HandleFunc("/deleteOneContact/{contact_numbers}", func(w http.ResponseWriter, r *http.Request) {
		var objOneContact OneContact
		objOneContact.Mob = gmux.Vars(r)["contact_numbers"]

		err := objOneContact.serveOneDelete(context)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}).Methods("DELETE")

	mux.HandleFunc("/css/{page_alias}",serveResource) //Ex: css/bootstrap.min.css
	mux.HandleFunc("/img/",serveResource)
	mux.HandleFunc("/js/" ,serveResource)


	n := negroni.Classic() //init negroni instance
	n.Use(sessions.Sessions("Web01",cookiestore.New([]byte("my-secret-123"))))
	//here we add a call to use our negroni middleware
	//we add our middleware before handler so it runs first
	//n.Use(negroni.HandlerFunc(objDbHandler.VerifyDatabase))
	n.UseHandler(mux)      //adding the multiplexer
	n.Run(":8080")

}

func (obj OneContact) ServeAddMoreContactNum(DBObj *DBHandler) (b Contacts, err error){

	result, err := DBObj.db.Exec("INSERT INTO numbers (name,contact_numbers) VALUES (?, ?) ",
		obj.Name, obj.Mob)

	x,_ := result.LastInsertId()
	fmt.Println("RESULT ONE ROW :" ,x)
	b = Contacts{
		Name: obj.Name,
		Mob:  obj.Mob,
	}
	return b, err
}

func (obj OneContact) ServeOneContact(DBObj *DBHandler) (P []OneContact, err error){

	rows, err:= DBObj.db.Query("select * from numbers where name=?", obj.Name)
	defer rows.Close()

	for rows.Next(){
		err = rows.Scan(&obj.Name, &obj.Mob)
		if err!=nil {
			return
		}
		P = append(P, obj)
	}
	return P, err

}


func (obj Contacts) serveAdd(DBObj *DBHandler) (b Contacts, err error) {

	result, err := DBObj.db.Exec("INSERT INTO contacts (name,mob,email,address,nationality,user_Name) VALUES (?, ?, ?,?,?,?) ",
		obj.Name, obj.Mob, obj.Email, obj.Address, obj.Nationality, obj.UserName)
	_,err1 := DBObj.db.Exec("insert into numbers (name,contact_numbers) values (?,?) ",obj.Name,obj.Mob)
	if err != nil || err1 !=nil {
		fmt.Println("ERROR Contacts/add :" + err.Error())
		return
	}

	x,_ := result.LastInsertId()
	fmt.Println("LAST INSERTED ROW: ",x)
	b = Contacts{
		Name: obj.Name,
		Mob:  obj.Mob,
		Email:obj.Email,
		Address:obj.Address,
		Nationality:obj.Nationality,
	}
	return b, err
}

func (obj User) serveRegister(DBObj *DBHandler) (error) {
	secret, _ := bcrypt.GenerateFromPassword([]byte(obj.Secret),bcrypt.DefaultCost)
	user := User{obj.UserName,secret} //new user object
	_, err := DBObj.db.Exec("INSERT INTO users (username,secret) VALUES (?, ?)", user.UserName, user.Secret)

	return err

}
func (obj User) serveLogin(DBObj *DBHandler) (p error, user *sql.Stmt, usr User)  {


		var us string
		us = obj.UserName

		user, err := DBObj.db.Prepare("select * from users where username =?")
		rows, err1 := user.Query(us);
		defer user.Close()
		if err1 != nil{
			p = err1
			fmt.Println("ERROR QUERY :", err1.Error())
			return
		}

		for rows.Next() {
			err := rows.Scan(&usr.UserName, &usr.Secret)
			if err!=nil {
				fmt.Println("ERRRR: ", err.Error())
				p = err
				return
			}
		}
		defer rows.Close()
		if err != nil {
			fmt.Println("ERROR USERS login: "+err.Error())
			p = err
			return
		}
		return p, user, usr

}

func (obj Contacts)serveContent(urlParameter map[string]string) (page_alias string, staticPage *template.Template) {

	//P := PContacts{Cont:[]Contacts{}}

	page_alias = urlParameter["page_alias"]

	if page_alias == "" {
		page_alias = "home"
		fmt.Println("1")
	}
	staticPage = populateStaticPages().Lookup(page_alias + ".html")
	if staticPage == nil {
		staticPage = populateStaticPages().Lookup("404.html")
	}
	return page_alias, staticPage
}
func (obj Contacts) serveContentTwo(DBObj *DBHandler)(P PContacts, err error){
	var str string
	str = obj.UserName
	u, err:= DBObj.db.Prepare("select * from contacts where user_Name=?") //u stands for users
	defer u.Close()
	rows, err := u.Query(str)

	for rows.Next(){
		//var b Contacts
		err = rows.Scan(&obj.Name, &obj.Mob, &obj.Email, &obj.Address, &obj.Nationality, &obj.UserName) //& so scan can edit the properties in memory

		P.Cont = append(P.Cont,obj)
	}
	defer rows.Close()
	return P, err
}

func (obj Contacts)serveDelete(DBObj *DBHandler) (err error){

	_,err = DBObj.db.Exec("delete from contacts where name = ?", obj.Name)

	//Tell the caller that everything is okay
	return err
}

func (obj OneContact)serveOneDelete(DBObj *DBHandler) (err error){
	//mob
	_,err = DBObj.db.Exec("delete from numbers where contact_numbers=?", obj.Mob)
	return err
}

//retrieve all files under subsequent page folders
func populateStaticPages() *template.Template {
	result := template.New("templates")
	templatePath := new ([]string)

	basePath:= "pages"
	templateFolder,_ :=os.Open(basePath)
	defer templateFolder.Close()
	templatePathRows,_ := templateFolder.Readdir(-1)
	for _,pathInfo := range templatePathRows {
		 log.Println(pathInfo.Name())
		*templatePath = append(*templatePath,basePath + "/" +pathInfo.Name())
	}
	result.ParseFiles(*templatePath...)
	return result
}

func getThemeName() string {
	return "bs4"
}

//to serve CSS files and apply it on site
func serveResource (w http.ResponseWriter, r *http.Request){
	path := "public/" + getThemeName() + r.URL.Path
	var contentType string

	if strings.HasSuffix(path,".css"){
		contentType = "text/css; charset=utf-8"
	}else if strings.HasSuffix(path,".png"){
		contentType = "image/png; charset=utf-8"
	}else if strings.HasSuffix(path,".jpg"){
		contentType = "image/jpg; charset=utf-8"
	}else if strings.HasSuffix(path,".is"){
		contentType = "application/javascript; charset=utf-8"
	}else {
		contentType = "text/plain; charset=utf-8"
	}
	log.Println(path)
	f,err := os.Open(path)
	fmt.Println("PATH IS :",path)
	if err == nil {
		defer f.Close()
		w.Header().Add("Content-Type",contentType)
		br:= bufio.NewReader(f)
		br.WriteTo(w)
		
	}else{
		w.WriteHeader(404)
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	     clearSession(w)
	     http.Redirect(w, r, "/login", http.StatusFound)

	 }

func clearSession(w http.ResponseWriter) {
	     cookie := &http.Cookie{
		         Name:   "session",
		         Value:  "",
		         Path:   "/",
		         MaxAge: -1,
		     }
	     http.SetCookie(w, cookie)
	 }

func main() {

	serveWeb()
}
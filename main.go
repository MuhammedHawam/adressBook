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

func main() {
	initDB()
	serveWeb()
}
var db *sql.DB //removed declaration here to easily access the variable from our middleware handler
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

var themeName = getThemeName() //yet to handle and from config file
var staticPages = populateStaticPages() //collect all pages under pages folder

func initDB(){
	db, _ = sql.Open("mysql", "root:20121993@/addressBook") //open connection with DB in MySql
}
func VerifyDatabase(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if err := db.Ping(); err != nil {
		fmt.Println("ERROR DB:"+err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//next is how the negroni knows how to continue executing middleware or route handlers after middleware is finished
	next(w, r);

}
func serveWeb(){
	mux := gmux.NewRouter()
	var objUser User
	var objContact Contacts
	var objDefaultInfoPage Contacts
	var objDeleteContacts Contacts
	var objOneContact OneContact
	mux.HandleFunc("/",objDefaultInfoPage.serveContent)
	mux.HandleFunc("/{page_alias}", objDefaultInfoPage.serveContent).Methods("GET")
	mux.HandleFunc("/login",objUser.serveLogin).Methods("POST")
	mux.HandleFunc("/add",objContact.serveAdd).Methods("PUT")
	mux.HandleFunc("/add/{name}",objDeleteContacts.serveDelete).Methods("DELETE")
	mux.HandleFunc("/login", logoutHandler).Methods("GET")
	mux.HandleFunc("/home/{name}",objOneContact.ServeOneContact).Methods("GET")
	mux.HandleFunc("/home/{name}",objOneContact.ServeAddMoreContactNum).Methods("PUT")
	mux.HandleFunc("/add/{contact_numbers}",objOneContact.serveOneDelete).Methods("DELETEE")
	//mux.HandleFunc("/",objDefaultInfoPage.serveShowInfo).Methods("")

	mux.HandleFunc("/css/{page_alias}",serveResource) //Ex: css/bootstrap.min.css
	mux.HandleFunc("/img/",serveResource)
	mux.HandleFunc("/js/" ,serveResource)


	n := negroni.Classic() //init negroni instance
	n.Use(sessions.Sessions("Web01",cookiestore.New([]byte("my-secret-123"))))
	//here we add a call to use our negroni middleware
	//we add our middleware before handler so it runs first
	n.Use(negroni.HandlerFunc(VerifyDatabase))
	n.UseHandler(mux)      //adding the multiplexer
	n.Run(":8080")

}

func (obj OneContact) ServeAddMoreContactNum(w http.ResponseWriter, r *http.Request){
	fmt.Println("the name here :",gmux.Vars(r)["name"])
	fmt.Println("the num here :",r.FormValue("contact_numbers"))
	result, err := db.Exec("INSERT INTO numbers (name,contact_numbers) VALUES (?, ?) ",
		gmux.Vars(r)["name"], r.FormValue("contact_numbers"))
	if err!=nil {
		fmt.Println("ERR ONE CONT NUM :",err.Error())
		http.Error(w,err.Error(),http.StatusInternalServerError)
	}
	x,_ := result.LastInsertId()
	fmt.Println("RESULT ONE ROW :" ,x)
	b:= Contacts{
		Name: gmux.Vars(r)["name"], //hab3t el name sa7 f el beet
		Mob:  r.FormValue("contact_numbers"),
	}
	if err:= json.NewEncoder(w).Encode(b); err != nil {
		fmt.Println("ERRORRRRR : " + err.Error())
		http.Error(w,err.Error(),http.StatusInternalServerError)
	}

}

func (obj OneContact) ServeOneContact(w http.ResponseWriter, r *http.Request){
	P := []OneContact{}
	fmt.Println("One Contact: ",gmux.Vars(r)["name"])
	rows,err:= db.Query("select * from numbers where name=?",gmux.Vars(r)["name"])
	if err!=nil {
		http.Error(w,err.Error(),http.StatusInternalServerError)

	}
	for rows.Next(){
		err = rows.Scan(&obj.Name,&obj.Mob)
		if err!=nil {
			http.Error(w,err.Error(),http.StatusInternalServerError)
		}
		 P = append(P,obj)
	}

	fmt.Println("MOBILEEEE : ",P)
	//staticPages.Execute(w,P)
	/*b:= Contacts{
		Name: gmux.Vars(r)["name"],
		Mob:  obj.Mob,
	}*/
	if err:= json.NewEncoder(w).Encode(P); err != nil {
		fmt.Println("ERRORRRRR One contact: " + err.Error())
		http.Error(w,err.Error(),http.StatusInternalServerError)
	}
}


func (obj Contacts) serveAdd(w http.ResponseWriter, r *http.Request)  {
	fmt.Println("Did you come here ?")
	result, err := db.Exec("INSERT INTO contacts (name,mob,email,address,nationality,user_Name) VALUES (?, ?, ?,?,?,?) ",
		r.FormValue("name"), r.FormValue("mob"), r.FormValue("email"),r.FormValue("address"), r.FormValue("nationality"),sessions.GetSession(r).Get("username"))
	_,err1 := db.Exec("insert into numbers (name,contact_numbers) values (?,?) ",r.FormValue("name"),r.FormValue("mob"))
	if err != nil || err1 !=nil {
		fmt.Println("ERROR Contacts/add :" + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	fmt.Println("RESULT : ", result)
	x,_ := result.LastInsertId()
	fmt.Println("LAST INSERTED ROW: ",x)
	b:= Contacts{
		Name: r.FormValue("name"),
		Mob:  r.FormValue("mob"),
		Email:r.FormValue("email"),
		Address:r.FormValue("address"),
		Nationality:r.FormValue("nationality"),
	}
	if err:= json.NewEncoder(w).Encode(b); err != nil {
		fmt.Println("ERRORRRRR : " + err.Error())
		http.Error(w,err.Error(),http.StatusInternalServerError)
	}
}

func (obj User) serveLogin(w http.ResponseWriter, r *http.Request)  {
	fmt.Println("Did you SERVE LOGIN ?")
	var p LoginPage
	if r.FormValue("register") != ""{
		fmt.Println("hello From Register: ",r.FormValue("register"))
		//if we need to register we'll need a bcrypt hash from the given pass
		secret,_ := bcrypt.GenerateFromPassword([]byte(r.FormValue("password")),bcrypt.DefaultCost)
		s := string(secret[:])
		fmt.Println("SECRET PASS = ",s)
		user := User{r.FormValue("username"),secret} //new user object

		fmt.Println("hello from Train Register two :",r.FormValue("username"))
		fmt.Println("Let's Print user : ",user.UserName)
		fmt.Println("Let's print Pass : ",string(user.Secret))


		_, err := db.Exec("INSERT INTO users (username,secret) VALUES (?, ?)", user.UserName, user.Secret)
		if err != nil {
			fmt.Println("ERROR USER INSERT :" + err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		fmt.Println("I REDIRECTED TO / page")
		http.Redirect(w,r,"/login",http.StatusFound)
		return

		fmt.Println("Did You come here?")
	}else if r.FormValue("login") != "" {
		fmt.Println("hello from Train Register two :",r.FormValue("username"))
		//user, err := dbmap.Get(User{},r.FormValue("username"))
		//////////////////////////////////////////////////
		var usr User
		var us string
		us = r.FormValue("username")
		fmt.Println("USERNAME : ",us)
		var pa string

		pa = r.FormValue("password")
		fmt.Println("PASSWORD : ",pa)
		//fmt.Println("select * from users where secret =?",[]byte(pa))

		user,err := db.Prepare("select * from users where username =?")
		rows,err1 := user.Query(us);
		if err1!=nil{
			fmt.Println("ERROR QUERY :",err1.Error())
		}

		for rows.Next() {
			err := rows.Scan(&usr.UserName,&usr.Secret)
			if err!=nil {
				fmt.Println("ERRRR: ",err.Error())
			}
		}

		fmt.Println("USERR EQUAL :",rows)

		if err != nil {
			fmt.Println("ERROR USERS login:"+err.Error())
			p.Error = err.Error()
			http.Error(w,err.Error(),http.StatusInternalServerError)
			return
		}

		//////////////////////////////////////////////////
		fmt.Println("USERNAME : ",us)
		fmt.Println("PASSWORD : ",pa)

		fmt.Println("hello from Train Register two :",r.FormValue("username"))
		if err!= nil {
			fmt.Println("ERROR LOGIN #1:", err.Error())
			p.Error = err.Error()
		}else if user == nil {
			fmt.Println("ERROR LOGIN #:",err.Error())
			p.Error = "NO SUCH USER WITH USERNAME "+ r.FormValue("username")
		}else{//the user isn't nil we'll perform a hard cast on object returned from the database to access user object

			//RETURN NIL ON SUCCESS authentication AND ERROR ON failure
			fmt.Println("HASH: ",[]byte(usr.Secret))
			fmt.Println("PASS comp: ",[]byte(pa))
			if err = bcrypt.CompareHashAndPassword([]byte(usr.Secret),[]byte(r.FormValue("password"))); err != nil {
				//if the err isn't nil let's set the err property on the page object
				fmt.Println("ERROR LOGIN #3: ",err.Error())
				p.Error = err.Error()
			}else {
				sessions.GetSession(r).Set("username",r.FormValue("username"))
				fmt.Println("I REDIRECTED HERE IN / AGAIN")
				http.Redirect(w,r,"/home",http.StatusFound)
				return
			}
		}

	}

}

func (obj Contacts)serveContent(w http.ResponseWriter, r *http.Request)  {
	fmt.Println("Hello From ServeCONTENT")
	P := PContacts{Cont:[]Contacts{}}
	urlParameter := gmux.Vars(r)
	page_alias := urlParameter["page_alias"]
	if page_alias == "" {
		page_alias="home"
		fmt.Println("1")
	}
	staticPage := staticPages.Lookup(page_alias + ".html")
	if page_alias == "home" {
		fmt.Println("2")
		fmt.Println("USERNAME SESSION :",sessions.GetSession(r).Get("username"))
		var str string
		str = sessions.GetSession(r).Get("username").(string)
		fmt.Println("STR : ",str)
		u,err:= db.Prepare("select * from contacts where user_Name=?") //u stands for users
		rows,err:=u.Query(str)
		if err != nil {
			fmt.Println("ERROR ServeContent:"+err.Error())
			http.Error(w,err.Error(),http.StatusInternalServerError)
			return
		}
		fmt.Println("ROWS :", rows)
		fmt.Println("3")
		for rows.Next(){
			//var b Contacts
			err := rows.Scan(&obj.Name, &obj.Mob, &obj.Email,&obj.Address, &obj.Nationality, &obj.UserName) //& so scan can edit the properties in memory
			if err!=nil {
				fmt.Println("ERRRR: ",err.Error())
			}
			fmt.Println("4")
			P.Cont = append(P.Cont,obj)
			fmt.Println("5")
		}

		fmt.Println("DATA :",P.Cont)
		fmt.Println("Before 6:",staticPage.Name())

		if len(P.Cont)>0 { //hya msh shaghala sa7 leh???????????????????????????

			fmt.Println("BOM")
			fmt.Println("AGAIN P: ",P.Cont)
			staticPage.Execute(w,P)

		}else{
			fmt.Println("CAROL")
			staticPage.Execute(w,nil)
		}

		fmt.Println("6")
	}else {
		fmt.Println("7")
		//fmt.Println("stat HERE: "+ staticPage.Name())
		fmt.Println("8")
		if staticPage == nil {
			staticPage = staticPages.Lookup("404.html")
			w.WriteHeader(404)
		}
		staticPage.Execute(w, nil)
	}
}

func (obj Contacts)serveDelete(w http.ResponseWriter, r *http.Request){
	fmt.Println("DELETE : ",gmux.Vars(r)["name"])
	if _,err:= db.Exec("delete from contacts where name = ?", gmux.Vars(r)["name"]); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//Tell the caller that everything is okay
	w.WriteHeader(http.StatusOK) //to send 200 status okay along with the response object when we return
}

func (obj OneContact)serveOneDelete(w http.ResponseWriter, r *http.Request){
	fmt.Println("DELETE one : ",gmux.Vars(r)["contact_numbers"])
	if _,err := db.Exec("delete from numbers where contact_numbers=?",gmux.Vars(r)["contact_numbers"]); err != nil {
		http.Error(w,err.Error(),http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
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
	path := "public/" + themeName + r.URL.Path
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
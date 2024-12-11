package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

var store = sessions.NewCookieStore([]byte("secret-key"))

type PageVariables struct {
	IsLoggedIn bool
	Username   string
}

type Work struct {
	ID            int    `json:"id"`
	Informate     string `json:"informate"`
	TimeDuration  int    `json:"time_duration"`
	StartDate     string `json:"start_date"`
	EndDate       string `json:"end_date"`
	Collaborators int    `json:"collaborators"`
	Username      string `json:"username"`
}

type Candidate struct {
	ID         int    `json:"id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Age        int    `json:"age"`
	Profession string `json:"profession"`
	Email      string `json:"email"`
	Module     int    `json:"module"`
}

func handleError(w http.ResponseWriter, err error, message string, statusCode int) {
	log.Println(message, err)
	http.Error(w, message, statusCode)
}

func validateFields(fields ...string) error {
	for _, field := range fields {
		if field == "" {
			return errors.New("все поля обязательны для заполнения")
		}
	}
	return nil
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	isLoggedIn := session.Values["username"] != nil

	var username string
	if isLoggedIn {
		username = session.Values["username"].(string)
	}

	pageVariables := PageVariables{
		IsLoggedIn: isLoggedIn,
		Username:   username,
	}

	tmpl, err := template.ParseFiles("web/home.html")
	if err != nil {
		handleError(w, err, "Ошибка загрузки шаблона", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, pageVariables)
	if err != nil {
		handleError(w, err, "Ошибка рендеринга шаблона", http.StatusInternalServerError)
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		if err := validateFields(username, password); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var storedPassword string
		err := GetDB().QueryRow("SELECT password FROM users WHERE username = ?", username).Scan(&storedPassword)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Неверные учетные данные", http.StatusUnauthorized)
			} else {
				handleError(w, err, "Ошибка базы данных", http.StatusInternalServerError)
			}
			return
		}

		if err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password)); err != nil {
			http.Error(w, "Неверные учетные данные", http.StatusUnauthorized)
			return
		}

		session, _ := store.Get(r, "session-name")
		session.Values["username"] = username
		session.Save(r, w)

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	http.ServeFile(w, r, "web/login.html")
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		if err := validateFields(username, password); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			handleError(w, err, "Ошибка хэширования пароля", http.StatusInternalServerError)
			return
		}

		_, err = GetDB().Exec("INSERT INTO users (username, password) VALUES (?, ?)", username, hashedPassword)
		if err != nil {
			if err.Error() == "UNIQUE constraint failed: users.username" {
				http.Error(w, "Пользователь уже существует", http.StatusConflict)
			} else {
				handleError(w, err, "Ошибка базы данных", http.StatusInternalServerError)
			}
			return
		}

		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	http.ServeFile(w, r, "web/register.html")
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	delete(session.Values, "username")
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func personalCabinetHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	if session.Values["username"] == nil {
		http.Redirect(w, r, "/personal_cabinet", http.StatusSeeOther)
		return
	}
	http.ServeFile(w, r, "web/personalcabinet.html")
}

func uslugi(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	if session.Values["username"] == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	http.ServeFile(w, r, "web/uslugi.html")
}

func candidatesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		rows, err := GetCandidatesDB().Query("SELECT id, first_name, last_name, age, profession, email, module FROM candidates")
		if err != nil {
			handleError(w, err, "Ошибка базы данных", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var candidates []Candidate
		for rows.Next() {
			var candidate Candidate
			if err := rows.Scan(&candidate.ID, &candidate.FirstName, &candidate.LastName, &candidate.Age, &candidate.Profession, &candidate.Email, &candidate.Module); err != nil {
				handleError(w, err, "Ошибка при извлечении данных", http.StatusInternalServerError)
				return
			}
			candidates = append(candidates, candidate)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(candidates); err != nil {
			handleError(w, err, "Ошибка при кодировании JSON", http.StatusInternalServerError)
			return
		}
		return
	}
}

func registerCandidateHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session-name")
	if err != nil {
		handleError(w, err, "Ошибка сессии", http.StatusInternalServerError)
		return
	}

	username, ok := session.Values["username"].(string)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		var exists bool
		err = GetCandidatesDB().QueryRow("SELECT EXISTS(SELECT 1 FROM candidates WHERE username = ?)", username).Scan(&exists)
		if err != nil {
			handleError(w, err, "Ошибка базы данных", http.StatusInternalServerError)
			return
		}

		if exists {
			http.Error(w, "Вы уже записались на экспедицию", http.StatusConflict)
			return
		}

		firstName := r.FormValue("first-name")
		lastName := r.FormValue("last-name")
		age := r.FormValue("age")
		profession := r.FormValue("profession")
		email := r.FormValue("email")
		module := r.FormValue("module")

		if err := validateFields(firstName, lastName, age, profession, email, module); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ageInt, err := strconv.Atoi(age)
		if err != nil {
			http.Error(w, "Возраст должен быть числом", http.StatusBadRequest)
			return
		}

		moduleInt, err := strconv.Atoi(module)
		if err != nil {
			http.Error(w, "Модуль должен быть числом", http.StatusBadRequest)
			return
		}

		_, err = GetCandidatesDB().Exec("INSERT INTO candidates (first_name, last_name, age, profession, email, module, username) VALUES (?, ?, ?, ?, ?, ?, ?)",
			firstName, lastName, ageInt, profession, email, moduleInt, username)
		if err != nil {
			handleError(w, err, "Ошибка базы данных", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/candidates", http.StatusSeeOther)
		return
	}

	http.ServeFile(w, r, "web/register_candidate.html")
}

func addWorkHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		session, _ := store.Get(r, "session-name")
		username, ok := session.Values["username"].(string)
		if !ok {
			http.Error(w, "Необходима аутентификация", http.StatusUnauthorized)
			return
		}

		informate := r.FormValue("informate")
		timeDuration := r.FormValue("time-duration")
		startDate := r.FormValue("start-date")
		endDate := r.FormValue("end-date")
		collaborators := r.FormValue("collaborators")

		if err := validateFields(informate, timeDuration, startDate, endDate, collaborators); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		timeDurationInt, err := strconv.Atoi(timeDuration)
		if err != nil {
			http.Error(w, "Длительность должна быть числом", http.StatusBadRequest)
			return
		}

		collaboratorsInt, err := strconv.Atoi(collaborators)
		if err != nil {
			http.Error(w, "Количество участников должно быть числом", http.StatusBadRequest)
			return
		}

		_, err = GetWorks().Exec("INSERT INTO works (informate, time_duration, start_date, end_date, collaborators, username) VALUES (?, ?, ?, ?, ?, ?)",
			informate, timeDurationInt, startDate, endDate, collaboratorsInt, username)
		if err != nil {
			handleError(w, err, "Ошибка базы данных", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Работа успешно добавлена!"))
		return
	}
	http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
}

func worksHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	username, ok := session.Values["username"].(string)
	if !ok {
		http.Error(w, "Необходима аутентификация", http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodGet {
		rows, err := GetWorks().Query("SELECT id, informate, time_duration, start_date, end_date, collaborators, username FROM works WHERE username = ?", username)
		if err != nil {
			handleError(w, err, "Ошибка базы данных", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var works []Work
		for rows.Next() {
			var work Work
			if err := rows.Scan(&work.ID, &work.Informate, &work.TimeDuration, &work.StartDate, &work.EndDate, &work.Collaborators, &work.Username); err != nil {
				handleError(w, err, "Ошибка при извлечении данных", http.StatusInternalServerError)
				return
			}
			works = append(works, work)
		}

		if err := rows.Err(); err != nil {
			handleError(w, err, "Ошибка при итерации по строкам", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(works); err != nil {
			handleError(w, err, "Ошибка при кодировании JSON", http.StatusInternalServerError)
			return
		}
		return
	}

	http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
}

func deleteWorkHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		workID := r.URL.Query().Get("id")
		if workID == "" {
			http.Error(w, "ID работы не указан", http.StatusBadRequest)
			return
		}

		_, err := GetWorks().Exec("DELETE FROM works WHERE id = ?", workID)
		if err != nil {
			handleError(w, err, "Ошибка при удалении работы", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Работа успешно удалена"))
		return
	}

	http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
}
func updateWorkHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		workID := r.FormValue("id")
		informate := r.FormValue("informate")
		timeDuration := r.FormValue("time_duration")
		startDate := r.FormValue("start_date")
		endDate := r.FormValue("end_date")
		collaborators := r.FormValue("collaborators")

		if err := validateFields(informate, timeDuration, startDate, endDate, collaborators); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		timeDurationInt, err := strconv.Atoi(timeDuration)
		if err != nil {
			http.Error(w, "Длительность должна быть числом", http.StatusBadRequest)
			return
		}

		collaboratorsInt, err := strconv.Atoi(collaborators)
		if err != nil {
			http.Error(w, "Количество участников должно быть числом", http.StatusBadRequest)
			return
		}

		_, err = GetWorks().Exec("UPDATE works SET informate = ?, time_duration = ?, start_date = ?, end_date = ?, collaborators = ? WHERE id = ?",
			informate, timeDurationInt, startDate, endDate, collaboratorsInt, workID)
		if err != nil {
			handleError(w, err, "Ошибка базы данных", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Работа успешно обновлена!"))
		return
	}

	http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
}

func getWorkHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workID := vars["id"]

	var work Work
	err := GetWorks().QueryRow("SELECT id, informate, time_duration, start_date, end_date, collaborators FROM works WHERE id = ?", workID).Scan(&work.ID, &work.Informate, &work.TimeDuration, &work.StartDate, &work.EndDate, &work.Collaborators)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Работа не найдена", http.StatusNotFound)
			return
		}
		handleError(w, err, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(work); err != nil {
		handleError(w, err, "Ошибка при кодировании JSON", http.StatusInternalServerError)
		return
	}
}

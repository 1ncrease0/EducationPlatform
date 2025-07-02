# EducationPlatform

Simple realization of education platform api on golang

# Features

- **User Authentication & Authorization**  (JWT access & refresh, RBAC)
- **Course Management** (creating, publishing, and managing courses)
- **Lesson & Module Editor** (uploading, editing and viewing content, multimedia supported)
- **Progress tracking** (for different types of quizes)
- **Search system** (text based search using elasticsearch)
- **Course Rating** (users can mark favourite courses)

# Technologies

- Go (gin, сleanenv, slog)
- Postgres
- Elasticsearch
- MiniO
- Docker & Docker Compose



# API Endpoints List

### Public

| Method | Path                    | Description                 |
|--------|-------------------------|-----------------------------|
| GET    | /v1/status              | Health check                |
| POST   | /v1/auth/login          | Login                       |
| POST   | /v1/auth/register       | Register new user           |
| POST   | /v1/auth/refresh        | Refresh JWT token           |

---

### Authenticated (All Roles)

| Method | Path         | Description                  |
|--------|--------------|------------------------------|
| GET    | /v1/me       | Get current user information |

---

###  Courses — Public Access

| Method | Path                                | Description                     |
|--------|-------------------------------------|---------------------------------|
| GET    | /v1/courses                         | Get all course previews         |
| GET    | /v1/courses/:course_id/preview      | Get course by ID (preview)      |
| GET    | /v1/courses/:course_id/content      | Get course structure & lessons  |
| GET    | /v1/courses/:course_id/status       | Get current status of course    |

---

###  Courses — Author Only

| Method | Path                                                           | Description                         |
|--------|----------------------------------------------------------------|-------------------------------------|
| GET    | /v1/courses/my-courses                                         | Get list of author's own courses    |
| POST   | /v1/courses                                                    | Create new course                   |
| PATCH  | /v1/courses/:course_id/publish                                 | Publish a course                    |
| PATCH  | /v1/courses/:course_id/hide                                    | Hide a course                       |
| PUT    | /v1/courses/:course_id/logo                                    | Upload or update course logo        |
| POST   | /v1/courses/:course_id/create-module                           | Create a new module                 |
| POST   | /v1/courses/:course_id/create-lesson                           | Create a new lesson                 |
| DELETE | /v1/courses/:course_id/module/:module_id                       | Delete a module                     |
| DELETE | /v1/courses/:course_id/module/:module_id/lesson/:lesson_id    | Delete a lesson from module         |
| PATCH  | /v1/courses/:course_id/lessons/swap                            | Swap positions of two lessons       |
| PATCH  | /v1/courses/:course_id/modules/swap                            | Swap positions of two modules       |
| POST   | /v1/courses/:course_id/lesson/content                          | Add text content to lesson          |
| POST   | /v1/courses/:course_id/lesson/content/media                    | Upload media content to lesson      |
| GET    | /v1/courses/:course_id/lessons/:lesson_id                      | Get lesson details                  |

---

###  Courses — Client Only

| Method | Path                                             | Description                         |
|--------|--------------------------------------------------|-------------------------------------|
| POST   | /v1/courses/:course_id/subscribe                 | Subscribe to course                 |
| GET    | /v1/courses/subscriptions                        | List subscribed courses             |
| GET    | /v1/courses/lessons/:lesson_id                   | Get lesson detail                   |
| POST   | /v1/courses/lessons/:lesson_id/quiz/submit       | Submit quiz answers                 |
| GET    | /v1/courses/lessons/:lesson_id/quiz/result       | Get quiz result                     |
| POST   | /v1/courses/:course_id/star                      | Rate the course                     |
| DELETE | /v1/courses/:course_id/star                      | Remove rating                       |
| GET    | /v1/courses/rated-status                         | Get rated courses by current user   |



# ClassKeeper

ClassKeeper is a comprehensive, open-source school management system designed to streamline administrative and academic tasks. It provides a full-featured backend powered by Go and a clean, functional frontend using vanilla HTML, CSS, and JavaScript.

The system allows for managing schools, classes, subjects, users (admins, teachers, students, parents), schedules, attendance, grades, homework, and announcements. It also includes analytics, data export capabilities, and a system for linking parents to their children for progress monitoring.

## Features

- **User Management**: Multi-role system with support for Admins, Teachers, Students, and Parents.
- **School & Class Organization**: Manage schools, create classes, assign class teachers, and manage student enrollments.
- **Academic Management**: Define subjects and assign teachers to them.
- **Scheduling**: Create and manage detailed class schedules for different days of the week.
- **Attendance Tracking**: Mark student attendance (present, absent, late, excused) for lessons.
- **Electronic Journal**: Post grades for students, track performance, and calculate average scores.
- **Homework Assignments**: Create, assign, and track homework with due dates for each class and subject.
- **Announcements**: A centralized system for school-wide, role-specific, or class-specific announcements.
- **Parental Portal**: Link parents to students to allow them to monitor their children's grades, attendance, and homework.
- **Analytics & Reporting**: View statistics for the school, classes, teachers, and subjects. Generate reports on attendance and grades.
- **Data Export**: Export class grades, attendance, and full student reports to CSV format.
- **System Settings**: Configure school information and perform database backups.

## Tech Stack

- **Backend**:
    - **Language**: Go (Golang)
    - **Framework**: Gin
    - **Database**: GORM (with support for PostgreSQL and SQLite)
    - **Authentication**: JWT (JSON Web Tokens)

- **Frontend**:
    - HTML5
    - CSS3
    - Vanilla JavaScript

## Getting Started

Follow these instructions to get a copy of the project up and running on your local machine.

### Prerequisites

You need to have **Go** installed on your system. You can download it from the [official Go website](https://go.dev/dl/).

### Installation & Configuration

1.  **Clone the repository:**
    ```sh
    git clone https://github.com/Volsogin/ClassKeeper.git
    cd ClassKeeper
    ```

2.  **Configure Environment Variables:**
    Navigate to the `backend` directory and create a `.env` file by copying the example:
    ```sh
    cd backend
    cp .env.example .env
    ```
    Open the `.env` file and customize the settings. For a quick start with SQLite, you don't need to change anything. To use PostgreSQL, set `DB_TYPE=postgres` and fill in your database credentials.

3.  **Install Dependencies:**
    The project uses Go Modules to manage dependencies. They will be downloaded automatically when you first run the application. Alternatively, you can use the provided setup script.

    *For Windows:*
    ```bat
    setup.bat
    ```

    *For Linux/macOS:*
    ```sh
    cd backend
    go mod download
    ```

### Running the Application

You can use the provided start scripts or run the Go application directly.

*   **Using Start Scripts (Recommended):**
    These scripts handle navigating to the correct directory and running the server.

    *For Windows:*
    ```bat
    start.bat
    ```
    *For Linux/macOS:*
    ```sh
    chmod +x start.sh
    ./start.sh
    ```

*   **Running Manually:**
    ```sh
    cd backend/cmd/server
    go run main.go
    ```

The server will start, and the application will be available at **`http://localhost:8080`**.

### Populating with Test Data

The repository includes a script to fill the database with a large set of test data, including classes, students, teachers, parents, and a full schedule.

1.  After starting the server for the first time, a `classkeeper.db` file will be created in the `backend` directory. **Register a new school and an admin user** through the web interface first.

2.  Copy the `classkeeper.db` file into the scripts directory:
    *Windows:*
    ```bat
    copy backend\classkeeper.db backend\scripts\
    ```
    *Linux/macOS:*
    ```sh
    cp backend/classkeeper.db backend/scripts/
    ```

3.  Run the fill script:
    ```sh
    cd backend/scripts
    go run fill_db.go
    ```

4.  The script will populate the database for the school you created. All generated users will have the password: **`password123`**.

## API Overview

The backend exposes a RESTful API with the following main endpoint groups:

- `/api/auth`: User registration and login.
- `/api/schools`: Manage school information.
- `/api/users`: CRUD operations for users.
- `/api/classes`: Manage classes and student enrollment.
- `/api/subjects`: Manage subjects and teacher assignments.
- `/api/schedules`: Manage class schedules.
- `/api/attendance`: Mark and view student attendance.
- `/api/grades`: Manage student grades.
- `/api/homework`: Manage homework assignments.
- `/api/announcements`: Create and view announcements.
- `/api/analytics`: Get statistics and reports.
- `/api/export`: Export data to CSV.
- `/api/parents`: Link parents to students and view child data.
- `/api/settings`: Manage school settings and backups.

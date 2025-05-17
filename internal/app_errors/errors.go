package app_errors

import "errors"

var ErrUserExists = errors.New("user already exists")
var ErrUserNotFound = errors.New("user not found")
var ErrIncorrectPassword = errors.New("incorrect password")
var ErrTokenNotFound = errors.New("token not found")
var ErrTokenExpired = errors.New("token expired")
var ErrCourseNotFound = errors.New("course not found")
var ErrNotCourseAuthor = errors.New("you are not course author")
var ErrNotImage = errors.New("not image")
var ErrFileSize = errors.New("file size error")
var ErrImageNotFound = errors.New("image not found")
var ErrCourseNotPublished = errors.New("course not published")
var ErrDuplicateLesson = errors.New("lesson with this order already exists in the module")
var ErrDuplicateModule = errors.New("module with this order already exists in the module")
var ErrAlreadySubscribed = errors.New("user is already subscribed to course")
var ErrNotRated = errors.New("not rated")
var ErrAlreadyRated = errors.New("already rated")

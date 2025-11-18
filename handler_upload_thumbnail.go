package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}

	mediaType := header.Header.Get("Content-Type")

	// get video metadata
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to get video from database", err)
		return
	}

	// check if it belongs to the currently logged in user
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Access denied to video as the user is not the owner", err)
		return
	}

	// getch extension from mediatype
	ext := strings.Split(mediaType, "/")[1]
	// save thumbnail to assets directory
	fileName := fmt.Sprintf("%v.%v", videoIDString, ext)
	fileURL := filepath.Join(cfg.assetsRoot, fileName)

	imageFile, err := os.Create(fileURL)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to create file", err)
		return
	}
	defer imageFile.Close()

	// get image data
	if _, err := io.Copy(imageFile, file); err != nil {
		respondWithError(w, http.StatusBadRequest, "Errro while copying image data from stream", err)
		return
	}

	//update video thumbnail url
	fullPublicURL := fmt.Sprintf("http://localhost:%v/%v", cfg.port, fileURL)
	log.Println(fullPublicURL)
	video.ThumbnailURL = &fullPublicURL

	// update video
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Can't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}

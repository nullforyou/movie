package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const movieSetting string = "resources/storage/setting.json"
const movieStorage string = "resources/storage/movies.json"

type Setting struct {
	Player string   `json:"player"`
	Path   []string `json:"path"`
}

type StorageDir struct {
	Path string `json:"dir"`
}

type Movies struct {
	Total int     `json:"total"`
	Movie []Movie `json:"movie"`
}

type Movie struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Timestamp int64  `json:"timestamp"`
}

func success(c *gin.Context, data interface{}) {
	c.JSON(200, gin.H{
		"success": true,
		"message": "",
		"data":    data,
	})
}

func failed(c *gin.Context, message string) {
	c.JSON(200, gin.H{
		"success": false,
		"message": message,
	})
}

func main() {
	router := gin.Default()
	router.GET("/movies", movies)
	router.GET("/setting", getSetting)
	router.POST("/setting", saveSetting)
	router.POST("/reload", reloadMovies)
	router.POST("/play", play)
	router.POST("/clearing", clearing)

	router.LoadHTMLGlob("./resources/templates/*")
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})
	router.GET("/setting.html", func(c *gin.Context) {
		c.HTML(http.StatusOK, "setting.html", nil)
	})
	router.Static("/assets", "./resources/assets") //静态文件服务
	router.Run(":8080")
}

func loadSetting() Setting {
	var setting Setting
	jsonData, err := openFile(movieSetting)
	if err != nil {
		return setting
	}
	err = json.Unmarshal(jsonData, &setting)
	if err != nil {
		return setting
	}
	return setting
}

func getPaths() []string {
	setting := loadSetting()
	return setting.Path
}

func getMovies(movieName string) Movies {
	jsonData, err := openFile(movieStorage)
	if err != nil {
		return Movies{}
	}
	var movies Movies
	err = json.Unmarshal(jsonData, &movies)
	if movieName != "" {
		var searchMovies Movies
		for _, movie := range movies.Movie {
			if strings.Contains(movie.Name, movieName) {
				searchMovies.Movie = append(searchMovies.Movie, movie)
				searchMovies.Total++
			}
		}
		movies = searchMovies
	}
	if err != nil {
		return Movies{}
	}
	return movies
}

/** 读取文件的内容 */
func openFile(filePath string) ([]byte, error) {
	b := make([]byte, 0, 512)
	file, err := os.Open(filePath)
	if err != nil {
		return b, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)
	b, err = io.ReadAll(file)
	if err != nil {
		return b, err
	}
	return b, nil
}

func writeFile(filePath string, data string) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	n, _ := file.Seek(0, io.SeekEnd)
	_, err = file.WriteAt([]byte(data), n)
	defer file.Close()
	if err != nil {
		return err
	}
	return nil
}

/** 获取存储路径数据 */
func path(c *gin.Context) {
	paths := getPaths()
	success(c, paths)
}

/** 获取电影数据 */
func movies(c *gin.Context) {
	movieName, _ := c.GetQuery("movieName")
	movies := getMovies(movieName)
	success(c, movies)
}

func getSetting(c *gin.Context) {
	success(c, loadSetting())
}

func play(c *gin.Context) {
	setting := loadSetting()
	cmd := exec.Command(setting.Player, c.PostForm("movieDir"))
	cmd.Run()
	success(c, nil)
}

type formB struct {
	Player string   `json:"player"`
	Path   []string `json:"path"`
}

func saveSetting(c *gin.Context) {
	setting := Setting{
		Player: c.PostForm("player"),
	}
	for _, value := range c.PostFormMap("path") {
		if !pathExists(value) {
			failed(c, fmt.Sprintf("目录[%s]不存在", value))
			return
		}
		setting.Path = append(setting.Path, value)
	}
	jsonStr, _ := json.Marshal(setting)
	err := writeFile(movieSetting, string(jsonStr))
	if err != nil {
		failed(c, err.Error())
		return
	}
	success(c, nil)
}

/** 清理删除包含[torrent]后缀的文件 **/
func clearing(c *gin.Context) {
	paths := getPaths()
	for _, dir := range paths {
		err := clearPathFile(dir)
		if err != nil {
			failed(c, err.Error())
			return
		}
	}
	success(c, nil)
}

/** 重新加载文件目录里的电影 */
func reloadMovies(c *gin.Context) {
	paths := getPaths()
	var movies Movies
	for _, dir := range paths {
		err := getPathFile(dir, &movies)
		if err != nil {
			failed(c, err.Error())
			return
		}
	}
	jsonStr, _ := json.Marshal(movies)
	err := writeFile(movieStorage, string(jsonStr))
	if err != nil {
		failed(c, err.Error())
		return
	}
	success(c, nil)
}

func clearPathFile(dir string) error {
	err := filepath.Walk(dir, func(filename string, fi os.FileInfo, err error) error {
		if fi == nil {
			return nil
		}
		if fi.IsDir() {
			return nil
		}
		extension := filepath.Ext(filename)
		if extension == ".torrent" {
			os.Remove(strings.Replace(filename, "\\", "/", -1))
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

/** 倒序排序 **/
func puzzle(arr *[]int) {
	for i := 0; i < len(*arr)-1; i++ {
		for j := 0; j < len(*arr)-1-i; j++ {
			var tem = 0
			if (*arr)[j] < (*arr)[j+1] {
				tem = (*arr)[j]
				(*arr)[j] = (*arr)[j+1]
				(*arr)[j+1] = tem
			}
		}
	}
}

/*
* 判断切片中是否存在此元素 **/

func IsElementInSlice(slice []int, target int) bool {
	sort.Ints(slice)
	index := sort.SearchInts(slice, target)
	return index < len(slice) && slice[index] == target
}

/** 根据目录获取目录下所有的文件及其子文件，并追加到movies中
 */
func getPathFile(dir string, movies *Movies) error {
	MovieMap := make(map[int64]Movie)
	timestamp := make([]int, 0)
	err := filepath.Walk(dir, func(filename string, fi os.FileInfo, err error) error {
		if fi == nil {
			return nil
		}
		if fi.IsDir() {
			return nil
		}
		extension := filepath.Ext(filename)
		if extension == ".mp4" || extension == ".rmvb" || extension == ".avi" || extension == ".flv" || extension == ".mkv" {
			var movie Movie
			movie.Path = strings.Replace(filename, "\\", "/", -1)
			movie.Name = fi.Name()
			currentTimestamp := int(fi.ModTime().UnixMicro())

			if IsElementInSlice(timestamp, currentTimestamp) {
				currentTimestamp = currentTimestamp + 1
			}
			timestamp = append(timestamp, currentTimestamp)
			movie.Timestamp = int64(currentTimestamp)
			MovieMap[movie.Timestamp] = movie
		}
		return nil
	})

	puzzle(&timestamp)

	for _, value := range timestamp {
		if movie, ok := MovieMap[int64(value)]; ok {
			movies.Movie = append(movies.Movie, movie)
			movies.Total++
		}
	}

	if err != nil {
		return err
	}
	return nil
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

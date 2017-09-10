package main

import (
	"fmt"
	"log"
	"bufio"
	"strconv"
	"os"
	"io/ioutil"
	"strings"
	"path/filepath"
	"image"
	"./libs/disintegration/imaging"
	"./libs/hawx/img/levels"
	"./libs/hawx/img/channel"
	"./libs/anthonynsimon/bild/noise"
	"sync"
)	
// map с условиями обрезания фотографий
type FileAnnotations map[string][][]int

func main() {
	createWorkingDirs();
	var annotationRules = readAnnotations();
	imagesPreparation(annotationRules);	
}

// функция создает рабочие директории
func createWorkingDirs() {
	err := os.Mkdir("fragments", os.FileMode(0522))
	err = os.Mkdir("fragments_flip", os.FileMode(0522))
	err = os.Mkdir("fragments_greyscale", os.FileMode(0522))
	err = os.Mkdir("fragments_noise", os.FileMode(0522))
	if err != nil {
		log.Print(err)
	}
}

// функция считывает содержимое директории images и в параллельных потоках производит над ними необходимые действия
func imagesPreparation(annotationRules FileAnnotations) {
	
	files, err := ioutil.ReadDir("./images")
	if err != nil {
		panic(err)
	}

	for _, f := range files {
		
		if (filepath.Ext(f.Name()) == ".png" || filepath.Ext(f.Name()) == ".jpg" || filepath.Ext(f.Name()) == ".jpeg") {
			src, err := imaging.Open("./images/" + f.Name())			
			if err != nil {
				panic(err)
			}			
		
			name := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))
			cropRulesArray := annotationRules[name]

  			wg := &sync.WaitGroup{}
			for i, cropRect := range cropRulesArray {
				wg.Add(1)
				go func(i int, cropRect []int) error {	
									
					proccessImage(name, i, src, cropRect, f)		
					
					wg.Done()
					return nil
				}(i, cropRect)
			}
        	wg.Wait()	

		}		
	}
}

// функция обрабатывает текущую фотографию и производит необходимые трансформации и фильтрации
func proccessImage(name string, i int, src image.Image, cropRect []int, f os.FileInfo) {
	
	rectangle := image.Rect(cropRect[0], cropRect[1], cropRect[2], cropRect[3])
	var cropedImg = imaging.Crop(src, rectangle)
	filePath := fmt.Sprintf("./fragments/%s_%d%s", name, i, filepath.Ext(f.Name()))
	extName := filepath.Ext(f.Name())
	err := imaging.Save(cropedImg, filePath)
	if err != nil {
		log.Fatal(err)
	}	
	
	var grayscaleImg = imaging.Grayscale(cropedImg)					
	err = imaging.Save(grayscaleImg, "./fragments_greyscale/"+name+"_"+ strconv.Itoa(i) +"_grey"+ extName)
	if err != nil {
		log.Fatal(err)
	}

	var flipedImg = imaging.Transpose(cropedImg)
	var autoLevelsImg = levels.Auto(flipedImg, channel.Brightness)
	err = imaging.Save(autoLevelsImg, "./fragments_flip/"+name+"_"+ strconv.Itoa(i) +"_flip"+ extName)
	if err != nil {
		log.Fatal(err)
	}

	var grayscaleImgBounds = grayscaleImg.Bounds()
	noiseImg := noise.Generate(grayscaleImgBounds.Dx(), grayscaleImgBounds.Dy(), &noise.Options{Monochrome: true, NoiseFn: noise.Gaussian})
	var gausianImg = imaging.Overlay(grayscaleImg, noiseImg, image.Pt(0, 0), 0.3)
	err = imaging.Save(gausianImg, "./fragments_noise/"+name+"_"+strconv.Itoa(i)+"_noise"+ extName)
	if err != nil {
		log.Fatal(err)
	}
}

// функция создает map условий обрезания фотографий
func readAnnotations() FileAnnotations {

	var annotationRules = make(map[string][][]int)

	files, err := ioutil.ReadDir("./annotations")
    if err != nil {
        log.Fatal(err)
    }

	for _, f := range files {
		file := constractFileAnnotation(f.Name())
		name := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))
		annotationRules[name] = file
    }

	return annotationRules

}

// функция обрабатывает файл с аннотациями и создает массив массивов для обрезания фотографии
func constractFileAnnotation(fileName string) ([][]int) {
	var concatsArray [][]int

	file, err := os.Open("annotations/" + fileName)	
	if err != nil {
    	log.Fatal(err)
    }
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		stringArray := strings.Split(scanner.Text(), ",")
		intArray, err := sliceAtoi(stringArray)
		if err != nil { 
			log.Fatal(err)
		}
		concatsArray = append(concatsArray, intArray)			
	}
	return concatsArray	
}

// функция парсит строку и создает из нее массив
func sliceAtoi(sa []string) ([]int, error) {
	si := make([]int, 0, len(sa))
	for _, a := range sa {
		i, err := strconv.Atoi(a)
		if err != nil {
			return si, err
		}
		si = append(si, i)
	}
	return si, nil
}
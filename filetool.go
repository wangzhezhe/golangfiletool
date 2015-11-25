package Filetool

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"sync"
	//"io/ioutil"
	"github.com/fsouza/go-dockerclient"
	"os"
	//"reflect"
	//"bytes"
	"bufio"
	"log"
	"os/exec"
	"strings"
	//"strings"
)

func Exist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

func CopyFile(source string, dest string) (err error) {

	sourcefile, err := os.Open(source)
	if err != nil {
		return err
	}

	defer sourcefile.Close()
	sourcemode, err := os.Stat(source)
	destfile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE, sourcemode.Mode())
	if err != nil {
		return err
	}

	defer destfile.Close()

	_, err = io.Copy(destfile, sourcefile)
	if err != nil {
		return err
	}
	return
}

func CopyDir(source string, dest string) (err error) {

	// get properties of source dir
	sourceinfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	// create dest dir

	err = os.MkdirAll(dest, sourceinfo.Mode())
	if err != nil {
		return err
	}

	directory, _ := os.Open(source)

	objects, err := directory.Readdir(-1)

	for _, obj := range objects {

		sourcefilepointer := source + "/" + obj.Name()

		destinationfilepointer := dest + "/" + obj.Name()

		if obj.IsDir() {
			// create sub-directories - recursively
			err = CopyDir(sourcefilepointer, destinationfilepointer)
			if err != nil {
				return err
			}
		} else {
			// perform copy
			err = CopyFile(sourcefilepointer, destinationfilepointer)
			if err != nil {
				return err
			}
		}
	}
	return
}

//compress the file (if compress to the zip file , using the similar package: zip.FileInfoHeader)
func Filecompress(tw *tar.Writer, dir string, fi os.FileInfo) error {

	//打开文件 open当中是 目录名称/文件名称 构成的组合
	log.Println(dir + fi.Name())
	fr, err := os.Open(dir + fi.Name())
	log.Println(fr.Name())
	if err != nil {
		return err
	}
	defer fr.Close()

	hdr, err := tar.FileInfoHeader(fi, "")

	hdr.Name = fr.Name()
	if err = tw.WriteHeader(hdr); err != nil {
		return err
	}

	_, err = io.Copy(tw, fr)
	if err != nil {
		return err
	}
	//打印文件名称
	log.Println("add the file: " + fi.Name())
	return nil

}

//compress the dir
func Dircompress(tw *tar.Writer, dir string) error {
	//打开文件夹
	dirhandle, err := os.Open(dir)
	//log.Println(dir.Name())
	//log.Println(reflect.TypeOf(dir))
	if err != nil {
		return err
	}
	defer dirhandle.Close()

	//fis, err := ioutil.ReadDir(dir)
	fis, err := dirhandle.Readdir(0)
	//fis的类型为 []os.FileInfo
	//log.Println(reflect.TypeOf(fis))
	if err != nil {
		return err
	}

	//遍历文件列表 每一个文件到要写入一个新的*tar.Header
	//var fi os.FileInfo
	for _, fi := range fis {
		if fi.IsDir() {

			//			//如果再加上这段的内容 就会多生成一层目录
			//			hdr, err := tar.FileInfoHeader(fi, "")
			//			if err != nil {
			//				panic(err)
			//			}
			//			hdr.Name = fi.Name()
			//			err = tw.WriteHeader(hdr)
			//			if err != nil {
			//				panic(err)
			//			}

			newname := dir + fi.Name()
			log.Println("using dir")
			log.Println(newname)
			//这个样直接continue就将所有文件写入到了一起 没有层级结构了
			//Filecompress(tw, dir, fi)
			err = Dircompress(tw, newname+"/")
			if err != nil {
				return err
			}

		} else {
			//如果是普通文件 直接写入 dir 后面已经有了 /
			err = Filecompress(tw, dir, fi)
			if err != nil {
				return err
			}
		}

	}
	return nil

}

//compress a dir into a tar file
func Dirtotar(sourcedir string, tardir string, newimage string) error {
	//file write 在tardir目录下创建
	_, err := os.Stat(sourcedir)
	if err != nil {
		log.Println("please create the deploy dir")
		return err
	}
	fw, err := os.Create(tardir + "/" + newimage + ".tar.gz")
	//type of fw is *os.File
	//	log.Println(reflect.TypeOf(fw))
	if err != nil {
		return err

	}
	defer fw.Close()

	//gzip writer
	gw := gzip.NewWriter(fw)
	defer gw.Close()

	//tar write
	tw := tar.NewWriter(gw)
	defer tw.Close()
	//	log.Println(reflect.TypeOf(tw))
	//add the deployments contens
	//Dircompress(tw, "deployments/")
	err = Dircompress(tw, sourcedir+"/")
	if err != nil {
		return err
	}
	//	// add the dockerfile
	//	fr, err := os.Open("Dockerfile")

	//do not package the dockerfile individual
	//write into the dockerfile
	//fileinfo, err := os.Stat(tardir + "/" + newimage + "_Dockerfile")
	//fileinfo, err := os.Stat("./Dockerfile")
	//if err != nil {
	//	panic(err)

	//}
	//log.Println(reflect.TypeOf(os.FileInfo(fileinfo)))
	//dockerfile要单独放在根目录下 和其他archivefile并列
	//Filecompress(tw, "", fileinfo)

	log.Println("tar.gz packaging OK")
	return nil

}

//return a tar stream
func SourceTar(filename string) *os.File {
	//"tardir/deployments.tar.gz"
	fw, _ := os.Open(filename)
	//log.Println(reflect.TypeOf(fw))
	return fw

}

//the systemcall funciton
func Systemexec(s string) {
	cmd := exec.Command("/bin/sh", "-c", s)
	log.Println(s)
	out, err := cmd.StdoutPipe()
	go func() {
		o := bufio.NewReader(out)
		for {
			line, _, err := o.ReadLine()
			if err == io.EOF {
				break
			} else {
				log.Println(string(line))
			}
		}
	}()
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

//create the temp dir and return this dir
//input image name
func Createdir(imagename string) (string, error) {

	//if the file already exist , delete and recreate it
	exist := Exist(imagename)

	if exist {
		log.Println("the folder exist , remove it")
		Cleandir(imagename)
	}
	dirname := imagename
	err := os.MkdirAll(dirname, 0777)
	if err != nil {
		return "", err
	}
	log.Println("create succesful: " + dirname)
	return dirname, nil

}

//delete the dir recursively
func Cleandir(dirname string) {

	//打开文件夹
	dirhandle, err := os.Open(dirname)
	//log.Println(dirname)
	//log.Println(reflect.TypeOf(dir))
	if err != nil {
		panic(nil)
	}
	defer dirhandle.Close()

	//fis, err := ioutil.ReadDir(dir)
	fis, err := dirhandle.Readdir(0)
	//fis的类型为 []os.FileInfo
	//log.Println(reflect.TypeOf(fis))
	if err != nil {
		panic(err)
	}

	//遍历文件列表 每一个文件到要写入一个新的*tar.Header
	//var fi os.FileInfo
	for _, fi := range fis {
		if fi.IsDir() {
			newname := dirname + "/" + fi.Name()
			//log.Println("using dir")
			//log.Println(newname)
			//这个样直接continue就将所有文件写入到了一起 没有层级结构了
			//Filecompress(tw, dir, fi)
			Cleandir(newname)

		} else {
			//如果是普通文件 直接写入 dir 后面已经有了 /
			filename := dirname + "/" + fi.Name()
			log.Println(filename)
			err := os.Remove(filename)
			if err != nil {
				panic(err)
			}
			log.Println("delete " + filename)
		}

	}
	//递归结束 删除当前文件夹
	err = os.Remove(dirname)
	log.Println("delete " + dirname)
	if err != nil {
		panic(err)
	}

}

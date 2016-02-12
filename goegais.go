package main

import (
  "os"
  "fmt"
  "flag"
  "io/ioutil"
  "net/http"
  "strings"
  "log"
  "encoding/xml"
  "strconv"
)

var flag_destination_directory  = flag.String("destdir", "egais_data", "destination directory for store XML files")
var flag_get_xml                = flag.Bool("get_xml", true, "get XML files for out block")
var flag_delete                 = flag.Bool("delete", false, "delete queries and replies")
var flag_out_block              = flag.Bool("out_block", true, "work with /opt/out")
var flag_in_block               = flag.Bool("in_block", true, "work with /opt/in")
var flag_server_name            = flag.String("server_name", "http://192.168.1.26:8080", "full server path for EGAIS")
var flag_before_out_id          = flag.Int("max_id_out", 0, "maximum id for remove replies")
var flag_before_in_id           = flag.Int("max_id_in", 0, "maximum id for remove queries")

type Egais_url struct {
  ReplyId string `xml:"replyId,attr"`
  Path string `xml:",chardata"`
}

type Egais_A struct {
  Name xml.Name `xml:"A"`
  Urls []Egais_url `xml:"url"`
}

func fix_last_separator( data string ) string {
  last_pos := len(data) - 1
  if (data[last_pos] == '/') || (data[last_pos] == '\\') {
    return data[:last_pos]
  }
  return data
}

func get_server_name() string {
  return fix_last_separator( *flag_server_name )
}

func get_http_data( full_server_path string ) ([]byte, error) {
  var data []byte
  res, err := http.Get(full_server_path)
  if err != nil {
    log.Fatal(err)
    return data, err
  }

  out_content, err := ioutil.ReadAll( res.Body )
  res.Body.Close()
  if err != nil {
    log.Fatal(err)
    return data, err
  }

  return []byte(out_content), nil
}

func get_egais_list( full_server_path string) Egais_A {
  data := Egais_A{}

  bdata,err := get_http_data( full_server_path )
  if err != nil {
    log.Fatal(err)
    return data
  }
  err = xml.Unmarshal(bdata, &data)
  if err != nil {
    log.Fatal(err)
    return data
  }

  return data
}

func get_id_from_path( path string ) int {
  path_list := strings.Split( fix_last_separator(path), "/" )
  id_pos := len(path_list) - 1
  str_id := path_list[id_pos]
  id, err := strconv.Atoi(str_id)
  if err != nil {
    log.Fatal(err)
    return 0
  }

  return id
}

func get_filename_by_path( path string ) string {
  path_list := strings.Split( fix_last_separator(path), "/" )
  id_pos := len(path_list) - 1
  name_pos := len(path_list) - 2

  filename := fmt.Sprintf("%s_%s.xml", path_list[name_pos], path_list[id_pos])

  return filename
}

func save_file( dest_file string, data []byte ) {
  f,err := os.Create( dest_file )
  if err != nil {
    log.Fatal(err)
    return
  }

  _, err = f.Write( data )
  f.Close()
  if err != nil {
    log.Fatal(err)
    return
  }
}

func get_xml() {
  path_arr := []string{get_server_name(), "opt/out"}
  full_server_path := strings.Join(path_arr, "/")

  data := get_egais_list( full_server_path )
  dest_dir := *flag_destination_directory
  err := os.MkdirAll( dest_dir, 0777 )
  if err != nil {
    log.Fatal(err)
    return
  }

  for _,data_rec := range data.Urls {
    filename := get_filename_by_path( data_rec.Path )
    dest_file := strings.Join( []string{dest_dir, filename}, string(os.PathSeparator))

    // first file
    xml_data, err := get_http_data( data_rec.Path )
    if err != nil {
      log.Fatal(err)
      continue
    }
    save_file( dest_file, xml_data )

    if len(data_rec.ReplyId) != 0 {
      // second file
      dest_dir_reply := strings.Join( []string{dest_dir,data_rec.ReplyId}, string(os.PathSeparator))
      os.MkdirAll( dest_dir_reply, 0777 )
      dest_file_reply := strings.Join( []string{dest_dir_reply, filename}, string(os.PathSeparator))
      save_file( dest_file_reply, xml_data )
    }
  }
}

func send_delete ( path string ) {
  req, err := http.NewRequest("DELETE", path, nil)
  if err != nil {
    log.Fatal(err)
    return
  }
  resp, err := http.DefaultClient.Do(req)
  if err != nil {
    log.Fatal( err )
    return
  }
  resp.Body.Close()
}

func delete_egais( part string ) {
  full_server_path := strings.Join([]string{get_server_name(), "opt", part}, "/")
  fmt.Printf("Path: %s\n", full_server_path)
  data := get_egais_list( full_server_path )

  min_id := 0
  if part == "out" {
    min_id  = *flag_before_out_id
  } else if part == "in" {
    min_id  = *flag_before_in_id
  }

  for _, data_rec := range data.Urls {
    id := get_id_from_path( data_rec.Path )
    if id < min_id {
      send_delete( data_rec.Path )
    }
  }
}

func main () {
  flag.Parse()

  if (*flag_out_block && *flag_get_xml) {
    get_xml()
  }
  if (*flag_out_block && *flag_delete) {
    delete_egais("out")
  }
  if (*flag_in_block && *flag_delete) {
    delete_egais("in")
  }

  os.Exit(0)
}

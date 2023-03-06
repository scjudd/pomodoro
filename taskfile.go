package main

import (
	"io"
	"os"

	pomoprotos "github.com/scjudd/pomodoro/protos"
	"google.golang.org/protobuf/proto"
)

func save(filename string, tasklist []*task) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	protolist := &pomoprotos.TaskList{
		Items: make([]*pomoprotos.Task, 0, len(tasklist)),
	}

	for _, task := range tasklist {
		protolist.Items = append(protolist.Items, &pomoprotos.Task{
			Pomodoros:   int32(task.pomodoros),
			Completed:   int32(task.completed),
			Description: task.description,
		})
	}

	bytes, err := proto.Marshal(protolist)
	if err != nil {
		return err
	}

	_, err = f.Write(bytes)
	return err
}

func load(filename string) ([]*task, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	bytes, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	protolist := &pomoprotos.TaskList{}
	err = proto.Unmarshal(bytes, protolist)
	if err != nil {
		return nil, err
	}

	tasklist := make([]*task, 0, len(protolist.Items))
	for _, prototask := range protolist.Items {
		tasklist = append(tasklist, &task{
			pomodoros:   int(prototask.Pomodoros),
			completed:   int(prototask.Completed),
			description: prototask.Description,
		})
	}

	return tasklist, nil
}

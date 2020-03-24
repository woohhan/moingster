package controller

import (
	"github.com/woohhan/moingster/pkg/controller/kluster"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, kluster.Add)
}

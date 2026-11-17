//go:build !pins

package installer

// pinsLive indica que el binario de pruebas se compiló con la etiqueta pins y
// puede consultar el registro npm real (HCD-039, C-07).
const pinsLive = false

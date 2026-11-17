//go:build pins

package installer

// pinsLive indica que el binario de pruebas se compiló con la etiqueta pins y
// puede consultar el registro npm real (HCD-039, C-07). Con esta etiqueta solo
// se ejecutan TestPinsCheck y TestPinsValidate, siempre con -run.
const pinsLive = true

package transformerv2

// The main difference between mly/service/domain.Transform and mly/service/domain/transformerv2.Transform
// is that for domain.Transform the function needs to handle polymorphism between pure tensors and shared.Output
// transformerv2.Transform strictly is given tensors with the first dimension removed and handled ahead of time.

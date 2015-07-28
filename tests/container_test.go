package tests

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/fgrosse/goldi"
	"github.com/fgrosse/goldi/tests/testAPI"
)

var _ = Describe("Container", func() {
	var (
		registry  goldi.TypeRegistry
		config    map[string]interface{}
		container *goldi.Container
	)

	BeforeEach(func() {
		registry = goldi.NewTypeRegistry()
		config = map[string]interface{}{}
		container = goldi.NewContainer(registry, config)
	})

	It("should panic if a type can not be resolved", func() {
		Expect(func() { container.Get("foo.bar") }).To(Panic())
	})

	It("should resolve simple types", func() {
		registry.RegisterType("goldi.test_type", testAPI.NewMockType)
		Expect(container.Get("goldi.test_type")).To(BeAssignableToTypeOf(&testAPI.MockType{}))
	})

	It("should build the types lazily", func() {
		typeID := "goldi.test_type"
		generator := &testAPI.MockTypeFactory{}
		registry.RegisterType(typeID, generator.NewMockType)

		generatorWrapper, typeIsRegistered := registry[typeID]
		Expect(typeIsRegistered).To(BeTrue())
		Expect(generatorWrapper).NotTo(BeNil())

		Expect(generator.HasBeenUsed).To(BeFalse())
		container.Get(typeID)
		Expect(generator.HasBeenUsed).To(BeTrue())
	})

	It("should build the types as singletons (one instance per type ID)", func() {
		typeID := "goldi.test_type"
		generator := &testAPI.MockTypeFactory{}
		registry.RegisterType(typeID, generator.NewMockType)

		generatorWrapper, typeIsRegistered := registry[typeID]
		Expect(typeIsRegistered).To(BeTrue())
		Expect(generatorWrapper).NotTo(BeNil())

		firstResult := container.Get(typeID)
		secondResult := container.Get(typeID)
		thirdResult := container.Get(typeID)
		Expect(firstResult == secondResult).To(BeTrue())
		Expect(firstResult == thirdResult).To(BeTrue())
	})

	It("should pass static parameters as arguments when generating types", func() {
		typeID := "goldi.test_type"
		typeDef := goldi.NewType(testAPI.NewMockTypeWithArgs, "parameter1", true)
		registry.Register(typeID, typeDef)

		generatedType := container.Get("goldi.test_type")
		Expect(generatedType).NotTo(BeNil())
		Expect(generatedType).To(BeAssignableToTypeOf(&testAPI.MockType{}))

		generatedMock := generatedType.(*testAPI.MockType)
		Expect(generatedMock.StringParameter).To(Equal("parameter1"))
		Expect(generatedMock.BoolParameter).To(Equal(true))
	})

	It("should be able to use parameters as arguments when generating types", func() {
		typeID := "goldi.test_type"
		typeDef := goldi.NewType(testAPI.NewMockTypeWithArgs, "%parameter1%", "%parameter2%")
		registry.Register(typeID, typeDef)

		config["parameter1"] = "test"
		config["parameter2"] = true

		generatedType := container.Get("goldi.test_type")
		Expect(generatedType).NotTo(BeNil())
		Expect(generatedType).To(BeAssignableToTypeOf(&testAPI.MockType{}))

		generatedMock := generatedType.(*testAPI.MockType)
		Expect(generatedMock.StringParameter).To(Equal(config["parameter1"]))
		Expect(generatedMock.BoolParameter).To(Equal(config["parameter2"]))
	})

	It("should be able to inject already defined types into other types", func() {
		registry.Register("goldi.injected_type", goldi.NewType(testAPI.NewMockType))
		registry.Register("goldi.main_type", goldi.NewType(testAPI.NewTypeForServiceInjection, "@goldi.injected_type"))

		generatedType := container.Get("goldi.main_type")
		Expect(generatedType).NotTo(BeNil())
		Expect(generatedType).To(BeAssignableToTypeOf(&testAPI.TypeForServiceInjection{}))

		generatedMock := generatedType.(*testAPI.TypeForServiceInjection)
		Expect(generatedMock.InjectedType).To(BeAssignableToTypeOf(&testAPI.MockType{}))
	})

	It("should inject the same instance when it is used by different services", func() {
		registry.RegisterType("foo", testAPI.NewMockType)
		registry.RegisterType("type1", testAPI.NewTypeForServiceInjection, "@foo")
		registry.RegisterType("type2", testAPI.NewTypeForServiceInjection, "@foo")

		generatedType1 := container.Get("type1")
		generatedType2 := container.Get("type2")
		Expect(generatedType1).To(BeAssignableToTypeOf(&testAPI.TypeForServiceInjection{}))
		Expect(generatedType2).To(BeAssignableToTypeOf(&testAPI.TypeForServiceInjection{}))

		generatedMock1 := generatedType1.(*testAPI.TypeForServiceInjection)
		generatedMock2 := generatedType2.(*testAPI.TypeForServiceInjection)

		Expect(generatedMock1.InjectedType == generatedMock2.InjectedType).To(BeTrue(), "Both generated types should have the same instance of @foo")
	})

	It("should inject nil when using optional types that are not defined", func() {
		registry.Register("goldi.main_type", goldi.NewType(testAPI.NewTypeForServiceInjection, "@?goldi.optional_type"))

		generatedType := container.Get("goldi.main_type")
		Expect(generatedType).NotTo(BeNil())
		Expect(generatedType).To(BeAssignableToTypeOf(&testAPI.TypeForServiceInjection{}))

		generatedMock := generatedType.(*testAPI.TypeForServiceInjection)
		Expect(generatedMock.InjectedType).To(BeNil())
	})
})

# Constitución Técnica del Proyecto

Este documento extiende la Constitución Técnica canónica (v2.1.1) servida por GoMemory.

---

## Enmienda: Principios de Diseño Orientado a Objetos y Modular (SOLID)

**Fecha:** 2026-11-11  
**Estado:** Activa / Obligatoria  
**Motivo:** Enunciar formal y canónicamente los 5 principios de SOLID como estándar de diseño modular y desacoplamiento para todo el código del proyecto.

Todo código producido o modificado en este proyecto DEBE adherirse estrictamente a los cinco principios SOLID:

1. **Responsabilidad Única (Single Responsibility Principle - SRP):**  
   Cada clase, estructura, función o módulo DEBE tener una única responsabilidad y un solo motivo para cambiar. Se prohíben tipos o componentes multitarea (por ejemplo, mezclar acceso a I/O, persistencia o HTTP dentro de entidades de dominio).

2. **Abierto / Cerrado (Open/Closed Principle - OCP):**  
   Las entidades de software DEBEN estar abiertas para la extensión pero cerradas para la modificación. Nuevos comportamientos o integraciones deben incorporarse mediante nuevos adaptadores, implementaciones o composición, sin alterar contratos de dominio consolidados.

3. **Sustitución de Liskov (Liskov Substitution Principle - LSP):**  
   Los objetos o estructuras que implementen una interfaz o tipo base DEBEN poder sustituir a su tipo base sin romper el comportamiento esperado ni violar las precondiciones/postcondiciones asumidas por el llamador.

4. **Segregación de Interfaces (Interface Segregation Principle - ISP):**  
   Se DEBEN preferir múltiples interfaces específicas y acotadas antes que interfaces monolíticas de propósito general. Ningún cliente o consumidor debe verse forzado a depender de métodos que no utiliza.

5. **Inversión de Dependencias (Dependency Inversion Principle - DIP):**  
   Los módulos de alto nivel (dominio y casos de uso de aplicación) DEBEN depender de abstracciones (ports, interfaces), NUNCA de implementaciones concretas (adaptadores, drivers o infraestructura). Las implementaciones concretas se inyectan en el composition root.

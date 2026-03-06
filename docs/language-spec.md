## Comments

`// This is a single line comment.`

`/*
    This is a block comments.
*/`

`/*
    /* Nested block comments are not supported. */
*/`

## Package Declaration

`package "main"`

Must be declared first in a source file.
The package name must be a string literal.

## Functions

```
void main() {
}
```

You must define a main function as the entry in your program.

## Variables

`<type> <name>`

Examples:

```
int count

string name

int _underScoresAllowed

// This is invalid, variables cannot start with numbers:
int 1test
```

## Types

```
uint8
uint16
uint32
uint64
int8
int16
int32
int64
int (maps to int64)
string (immutable string literals)
float32
float64
float (maps to float64)
```

## User Types

```
struct Person {
    string name
    float height
}
```

## Control Flow
For Loops:
```
for <init>; <condition>; <increment> {
}

// Example
for int x; x < 10; x++ {
}
```

While Loops:
```
int a
while a < 10 {
    a++
}
```
The while condition can be any expression that resolves to a boolean.

`break` keyword will break out of loops.

`continue` keyword will jump to next iterations.

## Expressions

## Arithmetic

## Pointers

## Error Handling
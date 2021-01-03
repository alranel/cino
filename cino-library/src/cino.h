#ifndef CINO_H
#define CINO_H

#define TEST_PLAN(n)            \
    Serial.begin(9600);         \
    while (!Serial)             \
    {                           \
    }                           \
    Serial.print("{\"plan\":"); \
    Serial.print(n);            \
    Serial.println("}")

#define TEST_NOPLAN() TEST_PLAN(-1)

#define TEST_DONE() \
    Serial.println("{\"done\":true}")

void _cino_check(bool result, char *quoted_expr, char *file, int line, bool fatal)
{
    Serial.print("{\"result\":");
    Serial.print(result ? "true" : "false");
    Serial.print(",\"expr\":");
    Serial.print(quoted_expr);
    Serial.print(",\"file\":\"");
    String f(file);
    f.replace("\"", "");
    Serial.print(f.substring(f.lastIndexOf('/') + 1));
    Serial.print("\",\"line\":");
    Serial.print(line);
    if (!result)
    {
        Serial.print(",\"fatal\":");
        Serial.print(fatal ? "true" : "false");
    }
    Serial.println("}");
    if (fatal && !result)
        while (1)
        {
        }
}

#define _quote(x) #x
#define REQUIRE(expr) _cino_check((expr), _quote(#expr), __FILE__, __LINE__, 1)
#define CHECK(expr) _cino_check((expr), _quote(#expr), __FILE__, __LINE__, 0)

#endif
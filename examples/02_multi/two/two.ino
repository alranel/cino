#include <cino.h>

void setup()
{
    TEST_PLAN(2);

    CHECK("foo" == "foo");
    CHECK("foo" != "bar");
}

void loop() {}

#include <cino.h>
#include <Wire.h>

// This test is supposed to fail.

void setup()
{
  TEST_PLAN(4);

  Wire.setClock(100000);
  CHECK("foo" == "bar");
  REQUIRE(TWI0.MBAUD == 67);
  REQUIRE(TWI0.MBAUD == 68);
  REQUIRE(2 == 3);
}

void loop() {}

#include <string.h>
// Smoke test of libgalaxy as the Swift app uses it (make lib-test).
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include "libgalaxy.h"

int main(void) {
	galaxy_init(0, 3000, "de");
	galaxy_viewport(1440, 900);
	float *buf = malloc(sizeof(float) * 12 * 20000);
	GalaxyFrameInfo info;
	int frames = 0;
	for (int i = 0; i < 200 && frames == 0; i++) {
		galaxy_tick(60, 2, 0, 0);
		frames += galaxy_frame(buf, 20000, 0, &info);
		usleep(5000);
	}
	if (frames == 0 || info.bodies < 3000 || info.stars == 0) {
		fprintf(stderr, "no frame: %d bodies\n", info.bodies);
		return 1;
	}
	if (!galaxy_action("trails") || galaxy_action("nonsense")) {
		fprintf(stderr, "action names\n");
		return 1;
	}
	galaxy_set("mass", 2);
	galaxy_tick(60, 2, 0, 0);
	GalaxyState s;
	galaxy_state(&s);
	if (!(s.fx & 8) || s.mass != 2 || s.renderScale != 2) {
		fprintf(stderr, "state fx %d mass %g scale %g\n", s.fx, s.mass, s.renderScale);
		return 1;
	}
	char *name = galaxy_text("sceneName"), *topics = galaxy_text("topics");
	printf("ok: %d bodies, %d stars, scene %s, scale %g, first topic: %.30s\n", info.bodies, info.stars, name, s.renderScale, topics);
	galaxy_free(name);
	galaxy_free(topics);
	char *err = galaxy_open("/nonexistent.galaxy");
	if (err == NULL) {
		fprintf(stderr, "opening a missing file must fail\n");
		return 1;
	}
	galaxy_free(err);
	char *explained = galaxy_setting("c"), *none = galaxy_setting("nonsense");
	if (strstr(explained, "\"chart\"") == NULL || none[0] != 0) {
		fprintf(stderr, "setting info: %.80s / %s\n", explained, none);
		return 1;
	}
	galaxy_free(explained);
	galaxy_free(none);
	galaxy_show_topic("dust");
	galaxy_tick(60, 2, 0, 0);
	galaxy_state(&s);
	if (!s.explain) {
		fprintf(stderr, "show topic did not open the learning panel\n");
		return 1;
	}
	return 0;
}

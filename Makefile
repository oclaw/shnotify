BUILD_DIR := build
SUBDIRS := shnotify shnotifyd

.PHONY: all $(SUBDIRS) clean

all: $(BUILD_DIR) $(SUBDIRS)

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

$(SUBDIRS):
	cd $@ && go build -o $@
	cp $@/$@ $(BUILD_DIR)/

clean:
	rm -rf $(BUILD_DIR)
	for dir in $(SUBDIRS); do \
		$(MAKE) -C $$dir clean; \
		rm -f $$dir/$$dir; \
	done

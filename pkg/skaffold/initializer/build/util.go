/*
Copyright 2020 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package build

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
)

func matchBuildersToImages(builders []InitBuilder, images []string) ([]BuilderImagePair, []InitBuilder, []string) {
	images = stripTags(images)

	var pairs []BuilderImagePair
	var unresolvedImages = make(sortedSet)
	for _, image := range images {
		builderIdx := findExactlyOneMatchingBuilder(builders, image)

		// exactly one builder found for the image
		if builderIdx != -1 {
			// save the pair
			pairs = append(pairs, BuilderImagePair{ImageName: image, Builder: builders[builderIdx]})
			// remove matched builder from builderConfigs
			builders = append(builders[:builderIdx], builders[builderIdx+1:]...)
		} else {
			// No definite pair found, add to images list
			unresolvedImages.add(image)
		}
	}
	return pairs, builders, unresolvedImages.values()
}

func findExactlyOneMatchingBuilder(builderConfigs []InitBuilder, image string) int {
	matchingConfigIndex := -1
	for i, config := range builderConfigs {
		if image != config.ConfiguredImage() {
			continue
		}
		// Found more than one match;
		if matchingConfigIndex != -1 {
			return -1
		}
		matchingConfigIndex = i
	}
	return matchingConfigIndex
}

func stripTags(taggedImages []string) []string {
	// Remove tags from image names
	var images []string
	for _, image := range taggedImages {
		parsed, err := docker.ParseReference(image)
		if err != nil {
			// It's possible that it's a templatized name that can't be parsed as is.
			warnings.Printf("Couldn't parse image [%s]: %s", image, err.Error())
			continue
		}
		if parsed.Digest != "" {
			warnings.Printf("Ignoring image referenced by digest: [%s]", image)
			continue
		}

		images = append(images, parsed.BaseName)
	}
	return images
}

func Artifacts(pairs []BuilderImagePair) []*latest.Artifact {
	var artifacts []*latest.Artifact

	for _, pair := range pairs {
		artifact := &latest.Artifact{
			ImageName: pair.ImageName,
		}

		workspace := filepath.Dir(pair.Builder.Path())
		if workspace != "." {
			fmt.Fprintf(os.Stdout, "using non standard workspace: %s\n", workspace)
			artifact.Workspace = workspace
		}

		pair.Builder.UpdateArtifact(artifact)

		artifacts = append(artifacts, artifact)
	}

	return artifacts
}

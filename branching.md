
## Branches

`main` - main development branch

`release/*` - branches for a release ready codebase with new features

`hotpatch/*` - branches for hotfixes and hot-change requests

`feat/*, fix/*, chore/*` - work branches

## Branch naming

Release of a signle service - `release/{current date}-{service name}` (e.g. `release/20220517-mpi`)

Release with multiple services - `release/{current date}` (e.g. `release/20220517`)

Hotfix, hotpatch - `hotpatch/{current date}-{service name}-{service version before patch}` (e.g. `release/20220517-mpi-v0.5.0`)

Work branch - `feat(fix, chore)/{service name}-{task number}-{short description}` (e.g. `feat/mpi-27-provide-logging`)

All the branches should consist of exactly two separated by `\` segments.

## Git tag naming

A tag should consist from the service name and service version according to [semver](https://semver.org/).
The version should meet the version from the `.semver.yaml` file of the service.

`{service name}-{version}` - e.g. `mpi-v0.5.0`


## Commit message

We use commit messages according to [conventions](https://www.conventionalcommits.org/en/v1.0.0/).

__Allowed commit types__ -  `feat, fix, chore`.

__Commit message title__ - `{commit type}[{scope}]: {message} (#{PR number})`.
For example `feat[service1,service2,pkg]:  provide HTTP requests logging (#13)`.
It is required to write a human readable, understandable message that will be clear for everyone.

__Commit message description__ - an additional commit info. Should be provided when it is unable to provide
all the necessary information in the commit title. Lines should not be too long.
Each new line should start with `** `. For example:
```
** improved the logging package to write fromatted logs
** used the new log formatting in the services
```

__Breaking change__ - it is required to add the [breaking change message](https://www.conventionalcommits.org/en/v1.0.0/#commit-message-with-description-and-breaking-change-footer) if the commit violates back compatibility

These requirements refers only commits that will appear in the final git history after `squashing`.
Local draft commit messages are up to you.

## Development process

- create a branch from the `main` branch. The branch's name should meet the [convention](`#branch-naming`).
- develop the feature in the created branch.
- create a pull request (PR) intended to merge the branch into the `main` branch.
- fill the PR's template.
- ensure that the lint and test processes (auto invoked github actions) are finished with success.
- start a code review process. The PR should be reviewed by at least two team members and be approved by all of them.
  All the conversations should be closed before merge. Conversations may be closed only by the author of the connversation.
- merge the branch via `squash and merge`. The commit message shoeld meet the [convention](#commit-message).
- delete the branch.
- ensure that all the CI processes (github actions) were successfully finished.


## Release process

- create and push a release branch from the `main` branch. The branch's name should meet the [convention](`#branch-naming`).
  All changes during release process should be made via PRs into the release branch.
  It is the desired way to create release branches via the [script](https://github.com/edenlabllc/wasfaty.api/blob/main/scripts/release-branch.sh). It will create and push the release branch. But if you have a special case you are free to create the release branch manually.
- create a work branch from the release branch.
- change service versions in `.semver.yaml` files according to [semver](https://semver.org/).
  It is also allowed to make other changes during the release preparation.
- make a PR to the release branch.
- merge the PR.
- wait until the CI process will create an intermediate build (e.g. v0.5.0-1a261c3-release).
- test the build, make potential fixes.
- create and push a git tag to the lates commit in the release branch according to the [convention](`#tag-naming`).
  It is the desired way to create tags via the [script](https://github.com/edenlabllc/wasfaty.api/blob/main/scripts/release-tag.sh). It will create and push the tag according to the [naming convention](`#tag-naming`) based on the service version
  form the service version from the `.semver.yaml` file. But if you have a special case you are free to create the tag manually.
  If there are several services in the release, you should create several tags (one per the service).
- pushed tag will initiate the github action that will create the final build (e.g. v0.5.0).
- create a merge branch from the `main` branch. Merge the release branch into the merge branch, fix conflicts.
  Make a PR to the `main`. Consider merge via `squash and merge` or `merge commit` based on the commits made
  to the release branch. If they are informative and you want to save them in the git hostiry, use `merge commit`.
- delete the release branch.

Note that it is possible to make a release from the head commit from the `main` branch or from some
certain commit from the `main` branch as well. Also if we want to release several services we are free
to make separate release branches for some (or every) of them. This gives us an opportunity
to choose a start point (head or one of previous commits) for the release branch for each service individually.
It doesn't impact the described above release flow.

## Urgent changes process

Urgent changes are both hot-fixes and change requests or any other code base changes that should be made
before an upcoming release. We'll use the `hot-patch` definition for such changes in future.
A concept of a `hot-patch` remains the same as for the [release](#release-process) except the several differences:
- a start point for the changes should be one of the previous stable releases (`git tags`).
  So a `hot-patch` branch should be created based on the appropriate `git tag`. It is the desired way
  to create `hot-patch` branches via the [script](https://github.com/edenlabllc/wasfaty.api/blob/main/scripts/release-branch.sh)
  by providing `--patch` flag. It will create and push the `hot-patch` branch. But if you have a special case you are free to create the release branch manually.
- only the patch version should be changed during the `hot-patch` flow (according to [semver](https://semver.org/)).
- the `hot-patch` branch should be merged into each of the upper versions. For example we have `v0.5.0`
  version in the `main` branch. Bug is found in the `v0.3.0`. We make the `hot-path` branch from the `v0.3.0`
  and make fixes. Tag the branch as `v0.3.1`. Create branches from the `v0.4.0 , v0.5.0` and merge fixes there,
  tag them as `v0.4.1 , v0.5.1`. Merge changes into the `main` branch.
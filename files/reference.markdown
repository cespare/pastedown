# Markdown Reference

Pastedown uses a [Markdown](http://daringfireball.net/projects/markdown/syntax)
variant similar to
[Github's](http://github.github.com/github-flavored-markdown/). This page shows
many of the supported features.

--------

## Text Formatting

    Text will automatically be rolled into paragraphs. You need two newlines
    to separate paragraphs -- single newlines will be ignored, to allow you
    to wrap the lines if you wish.

    Here is more text in a different paragraph.

Text will automatically be rolled into paragraphs. You need two newlines
to separate paragraphs -- single newlines will be ignored, to allow you
to wrap the lines if you wish.

Here is more text in a different paragraph.

    # This is a top-level header

    ## This is a second-level header

    #### This is a fourth-level header

# This is a top-level header

## This is a second-level header

#### This is a fourth-level header

Note that headers require a blank line afterwards before normal text resumes.

    Text can be made *italic*, **bold**, *or even **both** at once*.

Text can be made *italic*, **bold**, *or even **both** at once*.

--------

## Code

    Inline code can be made with backticks: `function() { return 3; }`.

Inline code can be made with backticks: `function() { return 3; }`.

Code blocks can be made by indenting the code four or more spaces beyond the
surrounding indentation level, or by surrounding the block with three or more
backticks:

        def answer():
          return 42

    ```
    def answer():
      return 42
    ```


These both display as:

    def answer():
      return 42

The second form also allows you to specify a language and have syntax
highlighting:

    ``` go
    func answer() int {
      return 42
    }
    ```

is rendered as:

``` go
func answer() int {
  return 42
}
```

--------

## Lists

Lists may be unordered or ordered, and may have sublists. Note that the
numbering is overridden for ordered lists.

    1.  Groceries
        * Milk
        * Eggs
        * Lettuce
    1.  Get car fixed
    1.  Optometrist appointment
        - Don't pass out in the waiting room this year

1.  Groceries
    * Milk
    * Eggs
    * Lettuce
1.  Get car fixed
1.  Optometrist appointment
    - Don't pass out in the waiting room this year

--------

## Links

    Links to websites like http://google.com are automatically created. You can
    also [make links explicitly](http://www.youtube.com/watch?v=oHg5SJYRHA0).

Links to websites like http://google.com are automatically created. You can also
[make links explicitly](http://www.youtube.com/watch?v=oHg5SJYRHA0).

--------

## Tables

You can make a table by drawing it:

    Name | Age
    -----|----
    Bob  | 45
    Sue  | 38

Name | Age
-----|----
Bob  | 45
Sue  | 38

--------

## Miscellaneous

Blockquotes:

    > Storm clouds on the mountains -- how truly beautiful they are! -- floating
    > fountains bearing water for every well; the angels of streams and lakes;
    > brooding in the deep pure azure, or sweeping along the ground, over ridge
    > and dome, over meadow, over forest, over garden and grove; lingering with
    > cooling shadows, refreshing every flower, and soothing rugged rock brows
    > with a gentleness of touch and gesture no human hand can equal!

> Storm clouds on the mountains -- how truly beautiful they are! -- floating
> fountains bearing water for every well; the angels of streams and lakes;
> brooding in the deep pure azure, or sweeping along the ground, over ridge and
> dome, over meadow, over forest, over garden and grove; lingering with cooling
> shadows, refreshing every flower, and soothing rugged rock brows with a
> gentleness of touch and gesture no human hand can equal!

-- John Muir

Inline images:

    ![Title](/public/shasta.jpg)

![Title](/public/shasta.jpg)
